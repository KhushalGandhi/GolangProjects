package main

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"sync"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"` // "admin" or "customer"
}

type Book struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Author        string    `json:"author"`
	PublishedDate time.Time `json:"published_date"`
	IsBorrowed    bool      `json:"is_borrowed"`
	BorrowedBy    string    `json:"borrowed_by"`
	BorrowedDate  time.Time `json:"borrowed_date"`
	ReturnDueDate time.Time `json:"return_due_date"`
}

var (
	users     = make(map[string]User)
	books     = make(map[string]Book)
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

	api.Post("/books", createBook)
	api.Get("/books", getBooks)
	api.Put("/books/:id", updateBook)
	api.Delete("/books/:id", deleteBook)
	api.Post("/borrow/:id", borrowBook)
	api.Post("/return/:id", returnBook)

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
	c.Locals("role", claims["role"])
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

	storedUser, exists := users[user.Username]
	if !exists || storedUser.Password != user.Password {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = user.Username
	claims["role"] = storedUser.Role
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not login"})
	}

	return c.JSON(fiber.Map{"token": t})
}

func createBook(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can create books"})
	}

	book := new(Book)
	if err := c.BodyParser(book); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	book.ID = uuid.New().String()
	books[book.ID] = *book

	return c.Status(fiber.StatusCreated).JSON(book)
}

func getBooks(c *fiber.Ctx) error {
	mutex.Lock()
	defer mutex.Unlock()

	var result []Book
	for _, book := range books {
		result = append(result, book)
	}

	return c.JSON(result)
}

func updateBook(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can update books"})
	}

	bookID := c.Params("id")
	book := new(Book)
	if err := c.BodyParser(book); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := books[bookID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "book not found"})
	}

	book.ID = bookID
	books[bookID] = *book

	return c.JSON(book)
}

func deleteBook(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can delete books"})
	}

	bookID := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := books[bookID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "book not found"})
	}

	delete(books, bookID)
	return c.SendStatus(fiber.StatusNoContent)
}

func borrowBook(c *fiber.Ctx) error {
	username := c.Locals("user").(string)
	bookID := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	book, exists := books[bookID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "book not found"})
	}

	if book.IsBorrowed {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "book is already borrowed"})
	}

	book.IsBorrowed = true
	book.BorrowedBy = username
	book.BorrowedDate = time.Now()
	book.ReturnDueDate = book.BorrowedDate.AddDate(0, 0, 14) // Borrow period is 14 days
	books[bookID] = book

	return c.JSON(book)
}

func returnBook(c *fiber.Ctx) error {
	username := c.Locals("user").(string)
	bookID := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	book, exists := books[bookID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "book not found"})
	}

	if !book.IsBorrowed || book.BorrowedBy != username {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "book is not borrowed by this user"})
	}

	book.IsBorrowed = false
	book.BorrowedBy = ""
	book.BorrowedDate = time.Time{}
	book.ReturnDueDate = time.Time{}
	books[bookID] = book

	return c.JSON(book)
}
