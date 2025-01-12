package DB

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" // Импорт драйвера для PostgreSQL
)

const (
	StatusAdded      = "Добавлено"
	StatusInProgress = "В процессе"
	StatusDone       = "Выполнено"
)

// addTaskToDb добавляет задачу в базу данных
func AddTaskToDb(db *sql.DB, t Task) error {
	status := StatusAdded
	t.Status = status // Устанавливаем статус "Добавлено" по умолчанию

	err := db.QueryRow(
		"INSERT INTO tasks (name, description, status, created_at) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		t.TaskName, t.Descr, t.Status, t.CreatedAt,
	).Scan(&t.ID, &t.CreatedAt) // Ожидаем два поля: ID и время создания
	if err != nil {
		log.Printf("Ошибка при добавлении задачи: %v", err)
		return fmt.Errorf("ошибка при добавлении задачи: %w", err)
	}

	fmt.Printf("Добавлена задача с ID: #%d\n", *t.ID)
	return nil
}

func WritingToChan(db *sql.DB, queue chan<- Task) error {
	rows, err := db.Query("SELECT id, name, description, status, created_at FROM tasks WHERE status = $1;", StatusAdded)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var t Task
		err := rows.Scan(&t.ID, &t.TaskName, &t.Descr, &t.Status, &t.CreatedAt)
		if err != nil {
			return fmt.Errorf("ошибка чтения строки: %w", err)
		}
		queue <- t
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("ошибка при итерации: %w", err)
	}

	return nil
}

func IsTaskIDUnique(db *sql.DB, taskID int) (bool, error) {
	var id sql.NullInt64
	query := "SELECT id FROM tasks WHERE id = $1"
	err := db.QueryRow(query, taskID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			// Если строка не найдена, значит ID уникален
			return true, nil
		}
		// Если другая ошибка, возвращаем ее
		return false, fmt.Errorf("ошибка проверки уникальности ID: %w", err)
	}
	// Если значение найдено, то ID не уникален
	return false, nil
}

// generateTasks генерирует список задач
func GenerateTasks(db *sql.DB, amount int) []Task { //??
	tasks := make([]Task, 0, amount)

	for i := 1; i <= amount; i++ {
		taskName := fmt.Sprintf("Задача #%d_%d", i, time.Now().UnixNano())
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM tasks WHERE name = $1)", taskName).Scan(&exists) //??
		if err != nil {
			log.Printf("Ошибка при проверке уникальности имени: %v", err)
			continue
		}

		if exists {
			log.Printf("Имя задачи '%s' уже существует, пропускаем", taskName)
			continue
		}

		task := Task{
			TaskName:  taskName,
			Descr:     fmt.Sprintf("Описание задачи #%d", i),
			Status:    StatusAdded,
			CreatedAt: time.Now(),
		}
		log.Printf("Задача с именем '%s' будет добавлена", taskName) // Логируем генерацию задачи
		tasks = append(tasks, task)
	}

	return tasks
}

func UpdateTaskStatus(db *sql.DB, taskID int, status string) error {
	_, err := db.Exec("UPDATE tasks SET status = $1 WHERE id = $2", status, taskID)
	if err != nil {
		return fmt.Errorf("Ошибка изменения статуса задачаи #%d: %w", taskID, err)
	}
	return nil
}
