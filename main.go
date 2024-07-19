package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"strconv"
	"sync"
	"time"
)

func main() {
	router := fiber.New()

}

type User struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
	Role     string `json:"role"` //will se later on
}

type Loan struct {
	Id         string `json:"id"`
	UserName   string `json:"user_name"`
	Amount     string `json:"amount"`
	Interest   string `json:"interest"`
	Tenure     string `json:"tenure"`
	MonthlyEmi string `json:"monthly_emi"`
	EmisPaid   string `json:"emis_paid"`
}

type Payment struct {
	UserName string `json:"user_name"`
	LoanId   string `json:"loan_id"`
	Amount   string `json:"amount"`
}


type Payme
// bantna seekh yaar
//
//
//dekh ek payment ka struct to bnana hi pdega tbhi aage uska use krke struct btnege na

// p* is affecting my ability to think basic cheezein bhi nhi soch paa rha

//

//var loans []Loan

var (
	loans = make(map[string]Loan)
	users = make(map[string]User)
	mutex = &sync.Mutex{}
)

func CreateLoan(c *fiber.Ctx) error {

	name := c.Params("customer_username")
	amount := c.Params("amount")

	amount1, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid amount format")
	}

	rate := c.Params("rate")

	rate1, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid amount format")
	}

	tenure := c.Params("tenure")

	tenure1, err := strconv.ParseFloat(tenure, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid amount format")
	}

	for _, user := range users {
		if user.UserName == name || user.Role == "admin" {
			return c.SendString("User does not have admin role")
		}
	}

	emi := CalculateEmi(amount1, rate1, tenure1)

}

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

func GetAllLoans(c *fiber.Ctx) error {

}

func GetLoanById(c *fiber.Ctx) error { // for customers

}

func GetAllLoanInfo(c *fiber.Ctx) error { // for admin

}

func CalculateEmi(amount float64, interest float64, tenure float64) float64 {
	return amount * interest * tenure / 100
}

func PayEMI(c *fiber.Ctx) error { // for customer

}
