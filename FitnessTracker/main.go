package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"log"
	"sync"
	"time"
)

func main() {
	router := fiber.New()

	router.Post("/signup", Register)
	router.Post("/login", LogIn)

	// Applying JWT middleware to the routes that need authentication
	api := router.Group("/api", jwtMiddleware)
	api.Post("/create/activity", CreateActivity)
	api.Post("/calculate/duration", ActivityHistory)
	api.Post("/start/:id", StartActivity)
	api.Post("/end/:user_activity_id", EndActivity)

	log.Fatal(router.Listen(":3000"))
}

type User struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
}

type Activity struct {
	ActivityId string `json:"activity_id"`
	Name       string `json:"name"`
}

type UserActivity struct {
	UserName       string `json:"user_name"`
	ActivityId     string `json:"activity_id"`
	UserActivityId string `json:"user_activity_id"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	Date           string `json:"date"`
}

type Return struct {
	UserName       string `json:"user_name"`
	ActivityId     string `json:"activity_id"`
	UserActivityId string `json:"user_activity_id"`
	Duration       string `json:"duration"`
	Date           string `json:"date"`
}

var (
	users          = make(map[string]User)
	activities     = make(map[string]Activity)
	useractivities = make(map[string]UserActivity)
	mutex          = &sync.Mutex{}
	jwtSecret      = []byte("secret")
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
	claims["user_id"] = storedUser.UserName
	claims["username"] = user.UserName
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
	return c.Next()
}

func CreateActivity(c *fiber.Ctx) error {
	activity := new(Activity)
	if err := c.BodyParser(activity); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	mutex.Lock()
	defer mutex.Unlock()

	activity.ActivityId = uuid.New().String()
	activities[activity.ActivityId] = *activity

	return c.Status(fiber.StatusCreated).JSON(activity)
}

func StartActivity(c *fiber.Ctx) error {
	username := c.Locals("username").(string)
	activityID := c.Params("id")

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := activities[activityID]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "activity not found"})
	}

	now := time.Now()

	formattedDate := now.Format("2006-01-02")
	formattedTime := now.Format("15:04:05")

	userActivity := UserActivity{
		UserActivityId: uuid.New().String(),
		UserName:       username,
		ActivityId:     activityID,
		Date:           formattedDate,
		StartTime:      formattedTime,
	}
	useractivities[userActivity.UserActivityId] = userActivity

	return c.Status(fiber.StatusCreated).JSON(userActivity)
}

func EndActivity(c *fiber.Ctx) error {
	username := c.Locals("username").(string)
	useractivityID := c.Params("user_activity_id")

	mutex.Lock()
	defer mutex.Unlock()

	userActivity, exists := useractivities[useractivityID]
	if !exists || userActivity.UserName != username {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no activity found with this ID for the user"})
	}

	endtime := time.Now()
	formattedTime := endtime.Format("15:04:05")

	userActivity.EndTime = formattedTime

	// ye cheez thi jahan jo maine socha bhi tha ki update krvana hai rather than vhi chlana hai to dhyan rkh is baat ka
	useractivities[useractivityID] = userActivity

	return c.Status(fiber.StatusOK).JSON(userActivity)
}

func ActivityHistory(c *fiber.Ctx) error {
	username := c.Locals("username").(string)

	mutex.Lock()
	defer mutex.Unlock()

	var result []Return
	for _, userActivity := range useractivities {
		if userActivity.UserName == username {
			baseModel := Return{
				UserName:       username,
				ActivityId:     userActivity.ActivityId,
				UserActivityId: userActivity.UserActivityId,
				Date:           userActivity.Date,
				Duration:       Duration(userActivity.StartTime, userActivity.EndTime),
			}
			result = append(result, baseModel)
		}
	}

	return c.JSON(result)
}

func Duration(startTimestr string, endTimestr string) string {
	layout := "15:04:05"

	startTime, err := time.Parse(layout, startTimestr)
	if err != nil {
		fmt.Println("Error parsing start time:", err)
		return ""
	}

	endTime, err := time.Parse(layout, endTimestr)
	if err != nil {
		fmt.Println("Error parsing end time:", err)
		return ""
	}

	if endTime.Before(startTime) {
		endTime = endTime.Add(24 * time.Hour)
	}

	difference := endTime.Sub(startTime)

	hours := int(difference.Hours())
	minutes := int(difference.Minutes()) % 60
	seconds := int(difference.Seconds()) % 60

	differenceStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	return differenceStr
}
