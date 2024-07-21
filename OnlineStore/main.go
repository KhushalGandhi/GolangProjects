package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"sync"
	"time"
)

func main() {
	router := fiber.New()

	router.Post("/signup", Register)
	router.Post("/login", LogIn)

	router.Post("/add", jwtMiddleware, AddItemsBySeller)
	router.Post("/delete/:id", jwtMiddleware, DeleteItemBySeller)
	router.Get("/view", ViewAllProducts) // Public route
	router.Post("/update/:id", jwtMiddleware, UpdateItemsBySeller)

	router.Get("/order/history", jwtMiddleware, OrderHistory)
	router.Post("/purchase/:id", jwtMiddleware, PurchaseProduct)

	router.Listen(":3000")
}

type User struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Password string `json:"password"`
	Role     string `json:"role"` // seller or buyer
}

type Product struct {
	ProductID   string `json:"product_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       string `json:"price"`
	Quantity    int    `json:"quantity"` // will see if change required later
}

type Purchase struct {
	PurchaseID   string    `json:"purchase_id"`
	UserID       string    `json:"user_id"`
	ProductID    string    `json:"product_id"`
	Quantity     int       `json:"quantity"`
	PurchaseDate time.Time `json:"purchase_date"`
}

var (
	users     = make(map[string]User)
	products  = make(map[string]Product)
	purchases = make(map[string]Purchase)
	mutex     = &sync.Mutex{} // make sure remember this is address
	jwtSecret = []byte("secret")
)

func Register(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := users[user.UserName]; exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user already exists"})
	}

	user.UserID = uuid.New().String()
	users[user.UserName] = *user
	return c.Status(fiber.StatusCreated).JSON(user)
}

func LogIn(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	storedUser, exists := users[user.UserName]
	if !exists || storedUser.Password != user.Password {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = storedUser.UserID
	claims["username"] = user.UserName
	claims["role"] = storedUser.Role
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not login"})
	}

	return c.JSON(fiber.Map{"token": t})
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

	c.Locals("user_id", claims["user_id"])
	c.Locals("username", claims["username"])
	c.Locals("role", claims["role"])
	return c.Next()
}

func AddItemsBySeller(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "seller" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only allowed for sellers"})
	}

	product := new(Product)
	if err := c.BodyParser(product); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	product.ProductID = uuid.New().String()
	products[product.ProductID] = *product

	return c.Status(fiber.StatusCreated).JSON(product)
}

func UpdateItemsBySeller(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "seller" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only allowed for sellers"})
	}

	productId := c.Params("id")
	product := new(Product)
	if err := c.BodyParser(product); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := products[productId]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}

	product.ProductID = productId
	products[productId] = *product

	return c.JSON(product)
}

func DeleteItemBySeller(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	if role != "seller" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only allowed for sellers"})
	}

	productId := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := products[productId]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}

	delete(products, productId)
	return c.SendStatus(fiber.StatusNoContent)
}

func ViewAllProducts(c *fiber.Ctx) error {
	mutex.Lock()
	defer mutex.Unlock()

	var result []Product
	for _, product := range products {
		result = append(result, product)
	}

	return c.JSON(result)
}

func PurchaseProduct(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	productID := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	product, exists := products[productID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}

	if product.Quantity == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "product out of stock"})
	}

	product.Quantity--
	products[productID] = product

	purchase := Purchase{
		PurchaseID:   uuid.New().String(),
		UserID:       userID,
		ProductID:    productID,
		Quantity:     1,
		PurchaseDate: time.Now(),
	}
	purchases[purchase.PurchaseID] = purchase

	return c.Status(fiber.StatusCreated).JSON(purchase)
}

func OrderHistory(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	mutex.Lock()
	defer mutex.Unlock()

	var result []Purchase
	for _, purchase := range purchases {
		if purchase.UserID == userID {
			result = append(result, purchase)
		}
	}

	return c.JSON(result)
}
