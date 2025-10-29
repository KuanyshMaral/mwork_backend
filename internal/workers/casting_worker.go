package workers

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"
)

type CastingWorker struct {
	db *gorm.DB
}

func NewCastingWorker(db *gorm.DB) *CastingWorker {
	return &CastingWorker{db: db}
}

// Start запускает фоновые задачи для кастингов
func (w *CastingWorker) Start(ctx context.Context) {
	// Автозакрытие просроченных кастингов каждый час
	go w.autoCloseCastings(ctx)
}

// autoCloseCastings автоматически закрывает кастинги с прошедшей датой
func (w *CastingWorker) autoCloseCastings(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Casting worker stopped")
			return
		case <-ticker.C:
			result := w.db.Exec(`
				UPDATE castings 
				SET status = 'closed', updated_at = NOW()
				WHERE status = 'active' 
				AND event_date < NOW()
			`)
			if result.Error != nil {
				log.Printf("Error auto-closing castings: %v", result.Error)
			} else if result.RowsAffected > 0 {
				log.Printf("Auto-closed %d expired castings", result.RowsAffected)
			}
		}
	}
}
