package main

import (
	"github.com/gofiber/fiber/v2"
	"sync"
)

func main() {
	router := fiber.New()

	books := router.Group("/books")
	{
		books.Post("/create") // these all routes are for admin because he only has the access to these things
		books.Get("/view")
		books.Put("/update") // this is for updation of details of the book
		books.Delete("/delete")
	}

	router.Post("/user/create", CreateUser)

	router.Post("/user/view") // this is for user

	router.Post("/user/update") // these two api's can be shown in a single way so he will know

}

var (
	users = make(map[string]User)
	books = make(map[string]int)
	mutex = &sync.Mutex{}
)

type User struct {
	UserName string `json:"user_name"`
	Type     string `json:"type"` // can be admin or user
}

type Book struct {
	Id           int32  `json:"id"`
	Title        string `json:"title"`
	Author       string `json:"author"`
	ISBN         string `json:"ISBN"`
	Availability string `json:"availability"` // we can also consider it as boolean
}

func CreateUser(c *fiber.Ctx) error {
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

func CreateBook(c *fiber.Ctx) error {
	adminUsername := c.Query("username")
	if user, ok := users[adminUsername]; !ok || user.Type != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	book := new(Book)
	if err := c.BodyParser(book); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	loanID := nextLoanID
	nextLoanID++

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"loan_id": loanID})
}

func ViewBook(c *fiber.Ctx) error {
	adminUsername := c.Query("username")
	if user, ok := users[adminUsername]; !ok || user.Type != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	var result []Book
	for _, loanID := range books[id] {
		loan := loans[loanID]
		result = append(result, loan)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}
