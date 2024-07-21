package main

import (
	"github.com/gofiber/fiber/v2"
	"log"
	_ "strconv"
	"sync"
)

type User struct {
	Username string `json:"username"`
	Type     string `json:"type"` // "admin" or "customer"
}

type Loan struct {
	CustomerUsername string  `json:"customer_username"`
	Principal        float64 `json:"principal"`
	InterestRate     float64 `json:"interest_rate"`
	Tenure           int     `json:"tenure"` // in years
	TotalAmount      float64 `json:"total_amount"`
	MonthlyEMI       float64 `json:"monthly_emi"`
	EMIsPaid         int     `json:"emis_paid"`
}

type Payment struct {
	Username string  `json:"username"`
	LoanID   int     `json:"loan_id"`
	Amount   float64 `json:"amount"`
}

var (
	users      = make(map[string]User)
	loans      = make(map[int]Loan)
	payments   = make(map[int][]Payment)
	userLoans  = make(map[string][]int)
	mutex      = &sync.Mutex{}
	nextLoanID = 1
)

func main() {
	app := fiber.New()

	app.Post("/create_user", createUser)
	app.Post("/create_loan", createLoan)
	app.Post("/make_payment", makePayment)
	app.Get("/loan_info", getLoanInfo)
	app.Get("/all_loans", getAllLoans)

	log.Fatal(app.Listen(":3000"))
}

func createUser(c *fiber.Ctx) error {
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

func createLoan(c *fiber.Ctx) error {
	adminUsername := c.Query("username")
	if user, ok := users[adminUsername]; !ok || user.Type != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	loan := new(Loan)
	if err := c.BodyParser(loan); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()
	if _, exists := users[loan.CustomerUsername]; !exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer does not exist"})
	}

	loanID := nextLoanID
	nextLoanID++

	I := (loan.Principal * float64(loan.Tenure) * loan.InterestRate) / 100
	A := loan.Principal + I
	loan.TotalAmount = A
	loan.MonthlyEMI = A / float64(loan.Tenure*12)
	loans[loanID] = *loan
	userLoans[loan.CustomerUsername] = append(userLoans[loan.CustomerUsername], loanID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"loan_id": loanID})
}

func makePayment(c *fiber.Ctx) error {
	username := c.Query("username")
	if _, exists := users[username]; !exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user does not exist"})
	}

	payment := new(Payment)
	if err := c.BodyParser(payment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()
	loan, exists := loans[payment.LoanID]
	if !exists || loan.CustomerUsername != username {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "loan not found or unauthorized access"})
	}

	loan.EMIsPaid++
	loans[payment.LoanID] = loan
	payments[payment.LoanID] = append(payments[payment.LoanID], *payment)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "payment successful"})
}

func getLoanInfo(c *fiber.Ctx) error {
	username := c.Query("username")
	if _, exists := users[username]; !exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user does not exist"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	var result []Loan
	for _, loanID := range userLoans[username] {
		loan := loans[loanID]
		result = append(result, loan)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func getAllLoans(c *fiber.Ctx) error {
	adminUsername := c.Query("username")
	if user, ok := users[adminUsername]; !ok || user.Type != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	var result []Loan
	for _, loan := range loans {
		result = append(result, loan)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}
