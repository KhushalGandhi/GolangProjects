package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"log"
	"sync"
	"time"
)

func main() {
	// Initialize Fiber app
	router := fiber.New()

	// Public routes
	router.Post("/signup", register)
	router.Post("/login", login)

	// Protected routes
	secured := router.Group("/", jwtMiddleware)

	// Admin routes
	admin := secured.Group("/admin", roleMiddleware("admin"))
	admin.Post("/job/add", CreateJobs)
	admin.Post("/job/remove/:application_id", RemoveJobs)
	admin.Post("/job/update/:application_id", UpdateJobs)

	// User routes
	user := secured.Group("/user", roleMiddleware("user"))
	user.Post("/application", ApplyingByUser)
	user.Get("/application/history/:user_name", ApplicationHistory)

	// Start the server
	log.Fatal(router.Listen(":3000"))
}

var (
	users            = make(map[string]User)
	applications     = make(map[string]Application)
	userapplications = make(map[string]UserApplication)
	mutex            = &sync.Mutex{}
	jwtSecret        = []byte("secret")
)

type User struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
	Role     string `json:"role"` // user or the company
}

type Application struct {
	ApplicationId string `json:"application_id"`
	Name          string `json:"name"`
	Position      string `json:"position"`
	Status        string `json:"status"`
	Date          string `json:"date"`
}

type UserApplication struct {
	UserName          string `json:"user_name"`
	UserApplicationId string `json:"user_application_id"`
	ApplicationId     string `json:"application_id"`
	Status            string `json:"status"`
	AppliedDate       string `json:"applied_date"`
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

	c.Locals("username", claims["username"])
	c.Locals("role", claims["role"])
	return c.Next()
}

func roleMiddleware(requiredRole string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("role").(string)
		if role != requiredRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		return c.Next()
	}
}

func register(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := users[user.UserName]; exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user already exists"})
	}

	users[user.UserName] = *user
	return c.Status(fiber.StatusCreated).JSON(user)
}

func login(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "error parsing"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	storedUser, exists := users[user.UserName]
	if !exists || storedUser.Password != user.Password {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = user.UserName
	claims["role"] = storedUser.Role
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not login"})
	}

	return c.JSON(fiber.Map{"token": t})
}

func CreateJobs(c *fiber.Ctx) error {
	application := new(Application)
	if err := c.BodyParser(application); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "error parsing"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	application.ApplicationId = uuid.New().String()
	applications[application.ApplicationId] = *application

	return c.Status(fiber.StatusCreated).JSON(application)
}

func RemoveJobs(c *fiber.Ctx) error {
	applicationId := c.Params("application_id")

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := applications[applicationId]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "application not found"})
	}

	delete(applications, applicationId)
	return c.SendStatus(fiber.StatusNoContent)
}

func UpdateJobs(c *fiber.Ctx) error {
	applicationID := c.Params("application_id")
	application := new(Application)
	if err := c.BodyParser(application); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := applications[applicationID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "application not found"})
	}

	application.ApplicationId = applicationID
	applications[applicationID] = *application

	return c.JSON(application)
}

func ApplyingByUser(c *fiber.Ctx) error {
	applicationId := c.Params("application_id")
	userName := c.Locals("username").(string)

	mutex.Lock()
	defer mutex.Unlock()

	application, exists := applications[applicationId]
	if !exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "application not present"})
	}

	if application.Status == "Applied" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "you have already applied for this job"})
	}

	baseModel := UserApplication{
		UserApplicationId: uuid.New().String(),
		ApplicationId:     applicationId,
		UserName:          userName,
		Status:            "Applied",
		AppliedDate:       CalculateDate(time.Now()),
	}

	userapplications[baseModel.UserApplicationId] = baseModel

	return c.Status(fiber.StatusCreated).JSON(baseModel)
}

func CalculateDate(time time.Time) string {
	now := time.Now()
	formattedDate := now.Format("2006-01-02")
	return formattedDate
}

func ApplicationHistory(c *fiber.Ctx) error {
	userName := c.Params("user_name")

	var result []UserApplication

	for _, application := range userapplications {
		if application.UserName == userName {
			result = append(result, application)
		}
	}

	return c.Status(fiber.StatusOK).JSON(result)
}
