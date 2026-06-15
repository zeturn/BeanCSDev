package k8s

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MySQLCredentialRuntime struct {
	Namespace      string
	ServiceName    string
	SecretName     string
	Database       string
	Username       string
	Password       string
	Port           string
	DependencyName string
	CredentialName string
}

func (m *Manager) ReconcileMySQLCredential(ctx context.Context, in MySQLCredentialRuntime) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if strings.TrimSpace(in.Namespace) == "" || strings.TrimSpace(in.ServiceName) == "" || strings.TrimSpace(in.SecretName) == "" {
		return fmt.Errorf("mysql dependency namespace, service, and secret are required")
	}
	if !safeMySQLIdentifier(in.Database) {
		return fmt.Errorf("mysql database %q must contain only letters, numbers, and underscores", in.Database)
	}
	if !safeMySQLAccount(in.Username) {
		return fmt.Errorf("mysql username %q contains unsupported characters", in.Username)
	}
	if in.Password == "" {
		return fmt.Errorf("mysql password is required")
	}
	port := strings.TrimSpace(in.Port)
	if port == "" || port == "<nil>" {
		port = "3306"
	}
	job := mysqlCredentialJob(in, port)
	created, err := m.Clientset.BatchV1().Jobs(in.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return m.waitForJob(ctx, created.Namespace, created.Name, 2*time.Minute)
}

func mysqlCredentialJob(in MySQLCredentialRuntime, port string) *batchv1.Job {
	backoffLimit := int32(1)
	ttl := int32(300)
	command := fmt.Sprintf(`for i in {1..90}; do
  if /opt/bitnami/mysql/bin/mysql -h %s -P %s -uroot -p"$MYSQL_ROOT_PASSWORD" -e 'SELECT 1' >/dev/null 2>&1; then
    mysql_ready=1
    break
  fi
  sleep 2
done
if [ "${mysql_ready:-}" != "1" ]; then
  echo "mysql service did not become ready"
  exit 1
fi
/opt/bitnami/mysql/bin/mysql -h %s -P %s -uroot -p"$MYSQL_ROOT_PASSWORD" <<'SQL'
CREATE DATABASE IF NOT EXISTS %s;
CREATE USER IF NOT EXISTS %s@'%%' IDENTIFIED BY %s;
ALTER USER %s@'%%' IDENTIFIED BY %s;
GRANT ALL PRIVILEGES ON %s.* TO %s@'%%';
FLUSH PRIVILEGES;
SQL
`, shellQuote(in.ServiceName), shellQuote(port), shellQuote(in.ServiceName), shellQuote(port), mysqlIdent(in.Database), mysqlString(in.Username), mysqlString(in.Password), mysqlString(in.Username), mysqlString(in.Password), mysqlIdent(in.Database), mysqlString(in.Username))
	labels := map[string]string{
		"managed-by":  "beancs",
		"beancs-task": "mysql-credential",
		"beancs-dep":  dnsLabel(in.DependencyName),
		"beancs-cred": dnsLabel(in.CredentialName),
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "beancs-mysql-cred-",
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
						Name:    "mysql-client",
						Image:   "docker.io/bitnamilegacy/mysql:9.4.0-debian-12-r1",
						Command: []string{"bash", "-ec", command},
						Env: []corev1.EnvVar{{
							Name: "MYSQL_ROOT_PASSWORD",
							ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: in.SecretName},
								Key:                  "mysql-root-password",
							}},
						}},
					}},
				},
			},
		},
	}
}

func (m *Manager) waitForJob(ctx context.Context, namespace, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		job, err := m.Clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if job.Status.Succeeded > 0 {
			return nil
		}
		if jobHasFailed(job) {
			return fmt.Errorf("job %s/%s failed", namespace, name)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for job %s/%s", namespace, name)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func jobHasFailed(job *batchv1.Job) bool {
	if job == nil {
		return false
	}
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

var mysqlIdentPattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
var mysqlAccountPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func safeMySQLIdentifier(value string) bool {
	return mysqlIdentPattern.MatchString(strings.TrimSpace(value))
}

func safeMySQLAccount(value string) bool {
	return mysqlAccountPattern.MatchString(strings.TrimSpace(value))
}

func mysqlIdent(value string) string {
	return "`" + strings.ReplaceAll(strings.TrimSpace(value), "`", "``") + "`"
}

func mysqlString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func dnsLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "unknown"
	}
	if len(value) > 63 {
		value = value[:63]
	}
	return strings.Trim(value, "-")
}
