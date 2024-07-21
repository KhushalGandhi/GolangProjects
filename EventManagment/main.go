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

	router.Post("/register", register)
	router.Post("/login", login)

	router.Post("/events", jwtMiddleware, createEvent)
	router.Get("/events", jwtMiddleware, getEvents)
	router.Put("/events/:id", jwtMiddleware, updateEvent)
	router.Delete("/events/:id", jwtMiddleware, deleteEvent)
	router.Post("/events/:id/book", jwtMiddleware, bookEventUser)
	router.Get("/bookings", jwtMiddleware, getUserBookings)

	router.Listen(":3000")
}

type User struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
	Type     string `json:"type"`
}

type Event struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Date        string   `json:"date"`
	Time        string   `json:"time"`
	Location    string   `json:"location"`
	BookedBy    []string `json:"booked_by"` // ye dekhne vaali baat hai ki ye optimal approach hai kya

	// ya to iska mtlb ye hai ki bs list hi ho rhi hai rather than alg se . try krke dekhta hun
	// to yahan username jaayega and eb yaar ye cheezein to terko settle hi honi chahiyein
}

var (
	users     = make(map[string]User)
	events    = make(map[string]Event)
	mutex     = &sync.Mutex{}
	jwtSecret = []byte("secret")
)

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
	claims["type"] = storedUser.Type
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not login"})
	}

	return c.JSON(fiber.Map{"token": t})
}

func createEvent(c *fiber.Ctx) error {
	role := c.Locals("type").(string)
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can create events"})
	}

	event := new(Event)
	if err := c.BodyParser(event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "error parsing"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	event.Id = uuid.New().String()
	event.BookedBy = []string{}
	events[event.Id] = *event

	return c.Status(fiber.StatusCreated).JSON(event)
}

func getEvents(c *fiber.Ctx) error {
	mutex.Lock()
	defer mutex.Unlock()

	var result []Event
	for _, event := range events {
		result = append(result, event)
	}

	return c.JSON(result)
}

func updateEvent(c *fiber.Ctx) error {
	role := c.Locals("type").(string)
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can update events"})
	}

	eventID := c.Params("id")
	event := new(Event)
	if err := c.BodyParser(event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := events[eventID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
	}

	event.Id = eventID
	events[eventID] = *event

	return c.JSON(event)
}

func deleteEvent(c *fiber.Ctx) error {
	role := c.Locals("type").(string)
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can delete events"})
	}

	eventID := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := events[eventID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
	}

	delete(events, eventID)
	return c.SendStatus(fiber.StatusNoContent)
}

func bookEventUser(c *fiber.Ctx) error {
	username := c.Locals("username").(string)
	eventID := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	event, exists := events[eventID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
	}

	for _, user := range event.BookedBy {
		if user == username {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user already booked this event"})
		}
	} // ab ye logic sochne vaala tha jb maine mandate ki api's bnayi thi to theek ho gya tha logic vgaera
	//vhi razorpay vaala dekhna chahta tha

	event.BookedBy = append(event.BookedBy, username)
	events[eventID] = event // and ab normal add kr diya events vaale list mein

	// ab ek doubt aur hai ki ye ki ismein db ka scene nhi hai na tbhi marshal and unmarshal ka koi scene nhi hai
	return c.JSON(event)
}

func getUserBookings(c *fiber.Ctx) error {
	username := c.Locals("username").(string)

	mutex.Lock()
	defer mutex.Unlock()

	var userBookings []Event
	for _, event := range events {
		for _, user := range event.BookedBy { // to ye to normal list reading hai
			if user == username {
				userBookings = append(userBookings, event)
			}
		}
	}

	return c.JSON(userBookings)
}
