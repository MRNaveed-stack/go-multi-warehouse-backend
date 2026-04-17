package utils

import (
	"log"
	"sync"
)

type LowStockPayLoad struct {
	ProductName     string
	CurrentQuantity int
}

type Job struct {
	Type string
	Data interface{}
}

var WG sync.WaitGroup
var JobQueue = make(chan Job, 100)

func StartWorkerPool(workerCount int) {
	for i := 0; i < workerCount; i++ {
		WG.Add(1)
		go func(id int) {
			defer WG.Done()
			for job := range JobQueue {
				log.Printf("Worker %d processing job: %s", id, job.Type)

				switch job.Type {
				case "LOW_STOCK_EMAIL":
					payload, ok := job.Data.(LowStockPayLoad)
					if !ok {
						log.Printf("Worker %d: Invalid payload for LOW_STOCK_EMAIL", id)
						continue
					}
					if err := SendLowStockAlert(payload.ProductName, payload.CurrentQuantity); err != nil {
						log.Printf("Worker %d error: %v", id, err)
					}
				case "WEBHOOK_SEND":
					payload, ok := job.Data.(WebhookJobData)
					if !ok {
						log.Printf("Worker %d: Invalid payload for WEBHOOK_SEND", id)
						continue
					}
					SendHttpRequest(payload.URL, payload.Payload, payload.Secret)
				default:
					log.Printf("Worker %d: Unknown job type %s", id, job.Type)
				}
			}
		}(i)
	}
}
