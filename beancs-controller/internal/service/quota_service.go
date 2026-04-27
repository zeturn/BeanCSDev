package service

import (
	"context"
	"fmt"

	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QuotaService struct {
	db *gorm.DB
}

func NewQuotaService(db *gorm.DB) *QuotaService { return &QuotaService{db: db} }

func (s *QuotaService) EnsureCanAllocate(ctx context.Context, teamID, preset string) error {
	spec, ok := model.ResourcePresets[preset]
	if !ok {
		return fmt.Errorf("unknown resource preset")
	}
	q, err := s.getOrCreate(ctx, teamID)
	if err != nil {
		return err
	}
	if q.UsedProjects+1 > q.MaxProjects {
		return fmt.Errorf("project quota exceeded")
	}
	if q.UsedCPUMillis+spec.CPUMillis > q.MaxCPUMillis {
		return fmt.Errorf("cpu quota exceeded")
	}
	if q.UsedMemoryMB+spec.MemoryMB > q.MaxMemoryMB {
		return fmt.Errorf("memory quota exceeded")
	}
	return nil
}

func (s *QuotaService) Reserve(ctx context.Context, teamID, preset string) error {
	spec, ok := model.ResourcePresets[preset]
	if !ok {
		return fmt.Errorf("unknown resource preset")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := &model.ResourceQuota{TeamID: teamID, MaxProjects: 10, MaxCPUMillis: 2000, MaxMemoryMB: 4096}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(q).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("team_id = ?", teamID).First(q).Error; err != nil {
			return err
		}
		if q.UsedProjects+1 > q.MaxProjects {
			return fmt.Errorf("project quota exceeded")
		}
		if q.UsedCPUMillis+spec.CPUMillis > q.MaxCPUMillis {
			return fmt.Errorf("cpu quota exceeded")
		}
		if q.UsedMemoryMB+spec.MemoryMB > q.MaxMemoryMB {
			return fmt.Errorf("memory quota exceeded")
		}
		return tx.Model(q).Updates(map[string]any{
			"used_projects":   q.UsedProjects + 1,
			"used_cpu_millis": q.UsedCPUMillis + spec.CPUMillis,
			"used_memory_mb":  q.UsedMemoryMB + spec.MemoryMB,
		}).Error
	})
}

func (s *QuotaService) Allocate(ctx context.Context, tx *gorm.DB, teamID, preset string) error {
	spec := model.ResourcePresets[preset]
	return tx.WithContext(ctx).Model(&model.ResourceQuota{}).Where("team_id = ?", teamID).
		Updates(map[string]any{
			"used_projects":   gorm.Expr("used_projects + ?", 1),
			"used_cpu_millis": gorm.Expr("used_cpu_millis + ?", spec.CPUMillis),
			"used_memory_mb":  gorm.Expr("used_memory_mb + ?", spec.MemoryMB),
		}).Error
}

func (s *QuotaService) ReleaseStandalone(ctx context.Context, teamID, preset string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.Release(ctx, tx, teamID, preset)
	})
}

func (s *QuotaService) Release(ctx context.Context, tx *gorm.DB, teamID, preset string) error {
	spec := model.ResourcePresets[preset]
	return tx.WithContext(ctx).Model(&model.ResourceQuota{}).Where("team_id = ?", teamID).
		Updates(map[string]any{
			"used_projects":   gorm.Expr("GREATEST(used_projects - ?, 0)", 1),
			"used_cpu_millis": gorm.Expr("GREATEST(used_cpu_millis - ?, 0)", spec.CPUMillis),
			"used_memory_mb":  gorm.Expr("GREATEST(used_memory_mb - ?, 0)", spec.MemoryMB),
		}).Error
}

func (s *QuotaService) getOrCreate(ctx context.Context, teamID string) (*model.ResourceQuota, error) {
	q := &model.ResourceQuota{TeamID: teamID, MaxProjects: 10, MaxCPUMillis: 2000, MaxMemoryMB: 4096}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(q).Error
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Where("team_id = ?", teamID).First(q).Error
	return q, err
}
