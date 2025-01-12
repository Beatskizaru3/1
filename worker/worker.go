package worker

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
	"workerpool/basic/DB"
)

func Worker(id int, db *sql.DB, tasks <-chan DB.Task, results chan<- DB.Task, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasks {
		log.Printf("Воркер #%d начал выполнение задачи #%d", id, *task.ID)

		// Здесь выполняется логика обработки задачи
		err := DB.UpdateTaskStatus(db, *task.ID, DB.StatusInProgress)
		if err != nil {
			log.Printf("Ошибка обновления статуса задачи #%d: %v", *task.ID, err)
			continue
		}

		// Имитация выполнения задачи (например, с задержкой)
		time.Sleep(time.Second * 3)
		fmt.Printf("Воркер #%d выполнил задачу #%d\n", id, *task.ID)

		err = DB.UpdateTaskStatus(db, *task.ID, DB.StatusDone)
		if err != nil {
			log.Printf("Ошибка обновления статуса задачи #%d: %v", *task.ID, err)
			continue
		}

		results <- task
	}
}
