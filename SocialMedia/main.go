package main

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var jwtSecret = []byte("secret")

type User struct {
	ID       string `gorm:"primaryKey"`
	Username string `gorm:"unique"`
	Password string
}

type Post struct {
	ID        string `gorm:"primaryKey"`
	Content   string
	Timestamp time.Time
	UserID    string
	User      User
}

type Follow struct {
	ID         string `gorm:"primaryKey"`
	FollowerID string // haan to isee pta lg jaayega na ki kis kisko follow kr rha hai
	FollowedID string
}

type Like struct {
	ID     string `gorm:"primaryKey"`
	PostID string // same in case of likes we can find the likeid through
	UserID string // userid se pta lg jaayega ki kaun kaunsi postId pe like mila hai
}

func ConnectDatabase() {
	dsn := "host=localhost user=postgres password=new+password dbname=social_media port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database: ", err)
	}

	DB = database

	err = DB.AutoMigrate(&User{}, &Post{}, &Follow{}, &Like{})
	if err != nil {
		log.Fatal("failed to migrate database: ", err)
	}
}

func main() {
	ConnectDatabase()

	app := fiber.New()

	app.Post("/register", register)
	app.Post("/login", login)

	app.Use(jwtMiddleware)

	app.Get("/users/:id", getUserProfile)
	app.Post("/users/:id/follow", followUser)
	app.Delete("/users/:id/unfollow", unfollowUser)

	app.Post("/posts", createPost)
	app.Get("/posts", getAllPosts)
	app.Get("/posts/:id", getPost)
	app.Patch("/posts/:id", updatePost)
	app.Delete("/posts/:id", deletePost)

	app.Post("/posts/:id/like", likePost)
	app.Delete("/posts/:id/unlike", unlikePost)

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

	c.Locals("userID", claims["userID"])
	return c.Next()
}

func register(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	user.ID = uuid.New().String()

	if err := DB.Create(user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not register user"})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

func login(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	var dbUser User
	if err := DB.Where("username = ? AND password = ?", user.Username, user.Password).First(&dbUser).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["userID"] = dbUser.ID
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not login"})
	}

	return c.JSON(fiber.Map{"token": t})
}

func getUserProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	var user User
	if err := DB.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return c.JSON(user)
}

func followUser(c *fiber.Ctx) error {
	followerID := c.Locals("userID").(string)
	followedID := c.Params("id")

	follow := Follow{
		ID:         uuid.New().String(),
		FollowerID: followerID,
		FollowedID: followedID,
	}

	if err := DB.Create(&follow).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not follow user"})
	}

	return c.Status(fiber.StatusCreated).JSON(follow)
}

func unfollowUser(c *fiber.Ctx) error {
	followerID := c.Locals("userID").(string)
	followedID := c.Params("id")

	if err := DB.Where("follower_id = ? AND followed_id = ?", followerID, followedID).Delete(&Follow{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not unfollow user"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func createPost(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	post := new(Post)
	if err := c.BodyParser(post); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	post.ID = uuid.New().String()
	post.UserID = userID
	post.Timestamp = time.Now()

	if err := DB.Create(post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create post"})
	}

	return c.Status(fiber.StatusCreated).JSON(post)
}

func getAllPosts(c *fiber.Ctx) error {
	var posts []Post

	if err := DB.Preload("User").Find(&posts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not retrieve posts"})
	}

	return c.JSON(posts)
}

func getPost(c *fiber.Ctx) error {
	id := c.Params("id")

	var post Post
	if err := DB.Preload("User").First(&post, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	return c.JSON(post)
}

func updatePost(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("userID").(string)

	var post Post
	if err := DB.First(&post, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not the owner of this post"})
	}

	if err := c.BodyParser(&post); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	if err := DB.Save(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update post"})
	}

	return c.JSON(post)
}

func deletePost(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("userID").(string)

	if err := DB.Where("id = ? AND user_id = ?", id, userID).Delete(&Post{}).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not the owner of this post"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func likePost(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	postID := c.Params("id")

	like := Like{
		ID:     uuid.New().String(),
		PostID: postID,
		UserID: userID,
	}

	if err := DB.Create(&like).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not like post"})
	}

	return c.Status(fiber.StatusCreated).JSON(like)
}

func unlikePost(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	postID := c.Params("id")

	if err := DB.Where("post_id = ? AND user_id = ?", postID, userID).Delete(&Like{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not unlike post"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
