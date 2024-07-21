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
	router := fiber.New()

	router.Post("/create", register)
	router.Post("/login", login)
	router.Use(jwtMiddleware) // Apply JWT middleware for all routes below
	router.Post("/blogs/create", CreateBlogs)
	router.Get("/blogs/view", GetAllBlogs)
	router.Get("/blog/view/:id", ViewBlog)       // Added :id parameter
	router.Post("/blog/delete/:id", DeleteBlogs) // Added :id parameter
	router.Post("/blog/update/:id", UpdateBlogs) // Added :id parameter

	log.Fatal(router.Listen(":3000"))

}

type User struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
}

type Blog struct {
	Id      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

var (
	users     = make(map[string]User)
	blogs     = make(map[string]Blog)
	mutex     = &sync.Mutex{}
	jwtSecret = []byte("secret")
)

func jwtMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid"})
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
	c.Locals("type", claims["type"])
	return c.Next()
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
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not login"})
	}

	return c.JSON(fiber.Map{"token": t})

}

func CreateBlogs(c *fiber.Ctx) error {
	blog := new(Blog)
	if err := c.BodyParser(blog); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	blog.Id = uuid.New().String()
	blogs[blog.Id] = *blog

	return c.Status(fiber.StatusCreated).JSON(blog)

}

func UpdateBlogs(c *fiber.Ctx) error {
	blogID := c.Params("id")
	blog := new(Blog)
	if err := c.BodyParser(blog); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := blogs[blogID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blog not found"})
	}

	blog.Id = blogID
	blogs[blogID] = *blog

	return c.JSON(blog)
}

func GetAllBlogs(c *fiber.Ctx) error {
	mutex.Lock()
	defer mutex.Unlock()

	var result []Blog

	for _, value := range blogs {
		result = append(result, value)
	}

	return c.JSON(result)
}

func DeleteBlogs(c *fiber.Ctx) error {

	blogID := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := blogs[blogID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blog not found"})
	}

	delete(blogs, blogID)
	return c.SendStatus(fiber.StatusNoContent)
}

func ViewBlog(c *fiber.Ctx) error {
	blogID := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := blogs[blogID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blog not found"})
	}

	return c.JSON(blogs[blogID])
}
