package k8s

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ExternalCredentialRuntime struct {
	Namespace       string
	Host            string
	Port            string
	Database        string
	Username        string
	Password        string
	AdminUsername   string
	AdminPassword   string
	ManagementPort  string
	DependencyName  string
	CredentialName  string
	DependencyType  string
	ControlledLabel string
}

func (m *Manager) ReconcileExternalMySQLCredential(ctx context.Context, in ExternalCredentialRuntime) error {
	if err := m.ensureExternalCredentialRuntime(in); err != nil {
		return err
	}
	if !safeMySQLIdentifier(in.Database) {
		return fmt.Errorf("mysql database %q must contain only letters, numbers, and underscores", in.Database)
	}
	if !safeMySQLAccount(in.Username) {
		return fmt.Errorf("mysql username %q contains unsupported characters", in.Username)
	}
	port := coalescePort(in.Port, "3306")
	command := fmt.Sprintf(`for i in {1..90}; do
  if mysql -h %s -P %s -u"$ADMIN_USERNAME" -p"$ADMIN_PASSWORD" -e 'SELECT 1' >/dev/null 2>&1; then
    mysql_ready=1
    break
  fi
  sleep 2
done
if [ "${mysql_ready:-}" != "1" ]; then
  echo "mysql service did not become ready"
  exit 1
fi
mysql -h %s -P %s -u"$ADMIN_USERNAME" -p"$ADMIN_PASSWORD" <<'SQL'
CREATE DATABASE IF NOT EXISTS %s;
CREATE USER IF NOT EXISTS %s@'%%' IDENTIFIED BY %s;
ALTER USER %s@'%%' IDENTIFIED BY %s;
GRANT ALL PRIVILEGES ON %s.* TO %s@'%%';
FLUSH PRIVILEGES;
SQL
`, shellQuote(in.Host), shellQuote(port), shellQuote(in.Host), shellQuote(port), mysqlIdent(in.Database), mysqlString(in.Username), mysqlString(in.Password), mysqlString(in.Username), mysqlString(in.Password), mysqlIdent(in.Database), mysqlString(in.Username))
	return m.runExternalCredentialJob(ctx, in, "mysql-client", "docker.io/mysql:8.4", command)
}

func (m *Manager) ReconcileExternalPostgreSQLCredential(ctx context.Context, in ExternalCredentialRuntime) error {
	if err := m.ensureExternalCredentialRuntime(in); err != nil {
		return err
	}
	if !safePostgresIdentifier(in.Database) {
		return fmt.Errorf("postgresql database %q must contain only letters, numbers, and underscores", in.Database)
	}
	if !safePostgresIdentifier(in.Username) {
		return fmt.Errorf("postgresql username %q must contain only letters, numbers, and underscores", in.Username)
	}
	port := coalescePort(in.Port, "5432")
	command := fmt.Sprintf(`export PGPASSWORD="$ADMIN_PASSWORD"
for i in {1..90}; do
  if psql -h %s -p %s -U "$ADMIN_USERNAME" -d postgres -tAc 'SELECT 1' >/dev/null 2>&1; then
    pg_ready=1
    break
  fi
  sleep 2
done
if [ "${pg_ready:-}" != "1" ]; then
  echo "postgresql service did not become ready"
  exit 1
fi
psql -h %s -p %s -U "$ADMIN_USERNAME" -d postgres -v ON_ERROR_STOP=1 -tc "SELECT 1 FROM pg_database WHERE datname = %s" | grep -q 1 || psql -h %s -p %s -U "$ADMIN_USERNAME" -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE %s"
psql -h %s -p %s -U "$ADMIN_USERNAME" -d postgres -v ON_ERROR_STOP=1 <<'SQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = %s) THEN
    EXECUTE format('CREATE ROLE %%I LOGIN PASSWORD %%L', %s, %s);
  ELSE
    EXECUTE format('ALTER ROLE %%I WITH LOGIN PASSWORD %%L', %s, %s);
  END IF;
END
$$;
SQL
psql -h %s -p %s -U "$ADMIN_USERNAME" -d %s -v ON_ERROR_STOP=1 -c "GRANT ALL PRIVILEGES ON DATABASE %s TO %s"
psql -h %s -p %s -U "$ADMIN_USERNAME" -d %s -v ON_ERROR_STOP=1 -c "GRANT ALL PRIVILEGES ON SCHEMA public TO %s"
`, shellQuote(in.Host), shellQuote(port), shellQuote(in.Host), shellQuote(port), postgresString(in.Database), shellQuote(in.Host), shellQuote(port), postgresIdent(in.Database), shellQuote(in.Host), shellQuote(port), postgresString(in.Username), postgresString(in.Username), postgresString(in.Password), postgresString(in.Username), postgresString(in.Password), shellQuote(in.Host), shellQuote(port), postgresIdent(in.Database), postgresIdent(in.Database), postgresIdent(in.Username), shellQuote(in.Host), shellQuote(port), postgresIdent(in.Database), postgresIdent(in.Username))
	return m.runExternalCredentialJob(ctx, in, "postgres-client", "docker.io/postgres:16", command)
}

func (m *Manager) ReconcileExternalRabbitMQCredential(ctx context.Context, in ExternalCredentialRuntime) error {
	if err := m.ensureExternalCredentialRuntime(in); err != nil {
		return err
	}
	port := coalescePort(in.ManagementPort, "15672")
	hostURL := "http://" + in.Host + ":" + port
	userPath := url.PathEscape(in.Username)
	command := fmt.Sprintf(`python3 - <<'PY'
import base64, json, os, time, urllib.error, urllib.request
base = %s
admin = os.environ["ADMIN_USERNAME"] + ":" + os.environ["ADMIN_PASSWORD"]
auth = "Basic " + base64.b64encode(admin.encode()).decode()
def request(method, path, body=None):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(base + path, data=data, method=method)
    req.add_header("authorization", auth)
    req.add_header("content-type", "application/json")
    with urllib.request.urlopen(req, timeout=10) as resp:
        resp.read()
for _ in range(90):
    try:
        request("GET", "/api/overview")
        break
    except Exception:
        time.sleep(2)
else:
    raise SystemExit("rabbitmq management API did not become ready")
request("PUT", "/api/users/%s", {"password": os.environ["CREDENTIAL_PASSWORD"], "tags": ""})
request("PUT", "/api/permissions/%%2F/%s", {"configure": ".*", "write": ".*", "read": ".*"})
PY
`, pythonString(hostURL), userPath, userPath)
	return m.runExternalCredentialJob(ctx, in, "rabbitmq-client", "docker.io/python:3.12-bookworm", command)
}

func (m *Manager) ensureExternalCredentialRuntime(in ExternalCredentialRuntime) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if strings.TrimSpace(in.Host) == "" {
		return fmt.Errorf("external dependency host is required")
	}
	if strings.TrimSpace(in.Username) == "" || strings.TrimSpace(in.Password) == "" {
		return fmt.Errorf("credential username and password are required")
	}
	if strings.TrimSpace(in.AdminUsername) == "" || strings.TrimSpace(in.AdminPassword) == "" {
		return fmt.Errorf("external dependency admin username and password are required")
	}
	return nil
}

func (m *Manager) runExternalCredentialJob(ctx context.Context, in ExternalCredentialRuntime, containerName, image, command string) error {
	if strings.TrimSpace(in.Namespace) == "" || in.Namespace == "<nil>" {
		in.Namespace = m.ControllerNamespace
	}
	if strings.TrimSpace(in.Namespace) == "" {
		in.Namespace = "beancs-system"
	}
	job := externalCredentialJob(in, containerName, image, command)
	created, err := m.Clientset.BatchV1().Jobs(in.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return m.waitForJob(ctx, created.Namespace, created.Name, 4*time.Minute)
}

func externalCredentialJob(in ExternalCredentialRuntime, containerName, image, command string) *batchv1.Job {
	backoffLimit := int32(1)
	ttl := int32(300)
	labels := map[string]string{
		"managed-by":  "beancs",
		"beancs-task": "external-credential",
		"beancs-dep":  dnsLabel(in.DependencyName),
		"beancs-cred": dnsLabel(in.CredentialName),
		"beancs-type": dnsLabel(in.DependencyType),
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "beancs-external-cred-",
			Namespace:    in.Namespace,
			Labels:       labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:    containerName,
						Image:   image,
						Command: []string{"bash", "-ec", command},
						Env: []corev1.EnvVar{
							{Name: "ADMIN_USERNAME", Value: in.AdminUsername},
							{Name: "ADMIN_PASSWORD", Value: in.AdminPassword},
							{Name: "CREDENTIAL_USERNAME", Value: in.Username},
							{Name: "CREDENTIAL_PASSWORD", Value: in.Password},
						},
					}},
				},
			},
		},
	}
}

func coalescePort(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "<nil>" {
		return fallback
	}
	return value
}

func safePostgresIdentifier(value string) bool {
	return mysqlIdentPattern.MatchString(strings.TrimSpace(value))
}

func postgresIdent(value string) string {
	return `"` + strings.ReplaceAll(strings.TrimSpace(value), `"`, `""`) + `"`
}

func postgresString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func pythonString(value string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`) + `"`
}
