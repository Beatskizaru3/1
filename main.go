package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"workerpool/basic/DB"
	"workerpool/basic/worker"

	_ "github.com/gorilla/mux"
	_ "github.com/lib/pq" // Импорт драйвера для PostgreSQL
)

type TaskRequest struct {
	TaskName string `json:"name"`
	Descr    string `json:"description"`
	Status   string `json:"status"`
}

func createTaskhandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "Неверный метод", http.StatusInternalServerError)
		return
	}
	var TaskReq TaskRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&TaskReq); err != nil {
		http.Error(w, "Ошибка в формате запроса", http.StatusBadRequest)
		return
	}

	if TaskReq.TaskName == "" || TaskReq.Descr == "" || TaskReq.Status == "" { //валидация введенных данных
		http.Error(w, "Отсутсвуют обязательные поля", http.StatusBadRequest)
		return
	}
	task := DB.Task{
		TaskName: TaskReq.TaskName,
		Descr:    TaskReq.Descr,
		Status:   TaskReq.Status,
	}

	err := DB.AddTaskToDb(db, task)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при добавлении задачи: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated) // 201 code
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)

}
func taskHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		http.Error(w, "Неверный метод", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT id, name, description, status, created_at FROM tasks;")
	if err != nil {
		http.Error(w, "Ошибка при получении задач", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []DB.Task
	for rows.Next() {
		var task DB.Task
		err := rows.Scan(&task.ID, &task.TaskName, &task.Descr, &task.Status, &task.CreatedAt)
		if err != nil {
			http.Error(w, "оишбка при обработке данных", http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Ошибка при обработке данных", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// понять почему не правильно происходит генерация айлишников для задач
// Реализация HTTP-сервера (REST API)
// Работа с JSON
func main() {

	// Подключение к базе данных
	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable password=Aezakmi1!")
	if err != nil {
		log.Fatalf("ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("ошибка проверки подключения к базе данных: %v", err)
	}
	// Генерация задач
	tasks := DB.GenerateTasks(db, 20)

	// Добавление задач в базу данных
	for _, task := range tasks { // Здесь используем tasks
		log.Printf("Добавление задачи: %v", task) // Логируем задачу перед добавлением
		if err := DB.AddTaskToDb(db, task); err != nil {
			log.Printf("ошибка добавления задачи: %v", err)
		}
	}
	// Обработчики
	http.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		taskHandler(w, r, db)
	})
	http.HandleFunc("/tasks/create", func(w http.ResponseWriter, r *http.Request) {
		createTaskhandler(w, r, db)
	})

	queue := make(chan DB.Task, 10)
	result := make(chan DB.Task, 10)
	wg := &sync.WaitGroup{} //??
	numWorkers := 3

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1) // ??
		go worker.Worker(i, db, queue, result, wg)
	}

	// Чтение задач из базы и отправка в канал
	go func() {
		err := DB.WritingToChan(db, queue)
		if err != nil {
			log.Printf("Ошибка записи задач в канал: %v", err)
		}
		close(queue) // Закрываем канал задач
	}()

	// Чтение завершенных задач
	go func() {
		for task := range result {
			fmt.Printf("Результат: задача #%d завершена\n", *task.ID)
		}
	}()

	wg.Wait()                                    // ждет выполнения всех горутин, которые относяться к wg
	log.Fatal(http.ListenAndServe(":8080", nil)) // Запуск сервера
}
