package basaltpass

import (
	"fmt"
	"sync"

	"github.com/zeturn/beancs-controller/internal/config"
	cryptoutil "github.com/zeturn/beancs-controller/internal/crypto"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

type ClientRegistry struct {
	mu         sync.RWMutex
	clients    map[uint]Client
	db         *gorm.DB
	cipher     cryptoutil.Cipher
	management Client
}

func NewClientRegistry(db *gorm.DB, cipher cryptoutil.Cipher, cfg *config.Config) *ClientRegistry {
	return &ClientRegistry{
		clients:    make(map[uint]Client),
		db:         db,
		cipher:     cipher,
		management: NewHTTPClient(cfg.BPMgmtBaseURL, cfg.BPMgmtClientID, cfg.BPMgmtClientSecret),
	}
}

func (r *ClientRegistry) GetManagementClient() (Client, error) {
	if r.management == nil {
		return nil, fmt.Errorf("management client not configured")
	}
	return r.management, nil
}

func (r *ClientRegistry) GetClientForInstance(instanceID uint) (Client, error) {
	r.mu.RLock()
	if client, ok := r.clients[instanceID]; ok {
		r.mu.RUnlock()
		return client, nil
	}
	r.mu.RUnlock()

	var inst model.BasaltPassInstance
	if err := r.db.First(&inst, instanceID).Error; err != nil {
		return nil, err
	}
	secret, err := r.cipher.DecryptString(inst.ClientSecretEnc)
	if err != nil {
		return nil, err
	}
	serviceToken := ""
	if len(inst.ServiceTokenEnc) > 0 {
		serviceToken, err = r.cipher.DecryptString(inst.ServiceTokenEnc)
		if err != nil {
			return nil, err
		}
	}
	client := NewHTTPClientWithServiceToken(inst.BaseURL, inst.ClientID, secret, serviceToken)

	r.mu.Lock()
	r.clients[instanceID] = client
	r.mu.Unlock()
	return client, nil
}

func (r *ClientRegistry) Invalidate(instanceID uint) {
	r.mu.Lock()
	delete(r.clients, instanceID)
	r.mu.Unlock()
}
