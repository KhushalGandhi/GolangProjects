package main

import (
	"github.com/golang-jwt/jwt/v4"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
	"sync"
)

type User struct {
	Username string `json:"username"` // maine userid rkha tha bs ye frk hai
	Password string `json:"password"`
}

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	DueDate     time.Time `json:"due_date"`
	Username    string    `json:"username"`
}

var (
	users     = make(map[string]User)
	tasks     = make(map[string]Task)
	userTasks = make(map[string][]string)
	mutex     = &sync.Mutex{}
	jwtSecret = []byte("secret")
)

func main() {
	app := fiber.New()

	// Middleware
	app.Use(logger.New())

	// Public routes
	app.Post("/register", register)
	app.Post("/login", login)

	// Restricted routes
	api := app.Group("/api", jwtMiddleware)

	api.Post("/tasks", createTask)
	api.Get("/tasks", getTasks)
	api.Put("/tasks/:id", updateTask)
	api.Delete("/tasks/:id", deleteTask)

	log.Fatal(app.Listen(":3000"))
}

func jwtMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or malformed JWT"})
	}

	tokenStr := authHeader[len("Bearer "):]
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired JWT"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid JWT claims"})
	}

	c.Locals("user", claims["username"])
	return c.Next()
}

func register(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := users[user.Username]; exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user already exists"})
	}

	users[user.Username] = *user
	return c.Status(fiber.StatusCreated).JSON(user)
}

func login(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	storedUser, exists := users[user.Username] // ye jo imememory hai
	if !exists || storedUser.Password != user.Password {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = user.Username
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not login"})
	}

	return c.JSON(fiber.Map{"token": t})
}

func createTask(c *fiber.Ctx) error {
	username := c.Locals("user").(string)
	task := new(Task)
	if err := c.BodyParser(task); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	task.ID = uuid.New().String()
	task.Username = username
	tasks[task.ID] = *task
	userTasks[username] = append(userTasks[username], task.ID)

	return c.Status(fiber.StatusCreated).JSON(task)
}

func getTasks(c *fiber.Ctx) error {
	username := c.Locals("user").(string)

	mutex.Lock()
	defer mutex.Unlock()

	var result []Task
	for _, taskID := range userTasks[username] {
		result = append(result, tasks[taskID])
	}

	return c.JSON(result)
}

func updateTask(c *fiber.Ctx) error {
	username := c.Locals("user").(string)
	taskID := c.Params("id")
	task := new(Task)
	if err := c.BodyParser(task); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	storedTask, exists := tasks[taskID]
	if !exists || storedTask.Username != username {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "task not found or unauthorized access"})
	}

	task.ID = taskID
	task.Username = username
	tasks[taskID] = *task

	return c.JSON(task)
}

func deleteTask(c *fiber.Ctx) error {
	username := c.Locals("user").(string)
	taskID := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	storedTask, exists := tasks[taskID]
	if !exists || storedTask.Username != username {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "task not found or unauthorized access"})
	}

	delete(tasks, taskID)
	for i, id := range userTasks[username] {
		if id == taskID {
			userTasks[username] = append(userTasks[username][:i], userTasks[username][i+1:]...)
			break
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}
