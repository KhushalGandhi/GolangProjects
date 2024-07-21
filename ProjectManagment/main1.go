package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"time"
)

type User struct {
	ID       string `gorm:"primaryKey" json:"id"`
	UserName string `gorm:"unique" json:"user_name"`
	Password string `json:"password"`
}

type Project struct {
	Id          string `gorm:"primaryKey" json:"id"`
	Description string `json:"description"`
	Name        string `json:"name"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	UserId      string `gorm:"index" json:"user_id"` // who created the project
}

type Task struct {
	Id          string `gorm:"primaryKey" json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	DueDate     string `json:"due_date"`
	ProjectId   string `gorm:"index" json:"project_id"`
	AssignedTo  string `gorm:"index" json:"assigned_to"`
}

type Team struct {
	Id   string `gorm:"primaryKey" json:"id"`
	Name string `json:"name"`
}

type TeamMember struct {
	Id     string `gorm:"primaryKey" json:"id"`
	TeamId string `gorm:"index" json:"team_id"`
	UserId string `gorm:"index" json:"user_id"`
}

var jwtSecret = []byte("secret")

var DB *gorm.DB

func ConnectDatabase() {
	dsn := "host=localhost user=postgres password=new+password dbname=social_media port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database: ", err)
	}

	DB = database

	err = DB.AutoMigrate(&User{}, &Team{}, &TeamMember{}, &Task{}, &Project{})
	if err != nil {
		log.Fatal("failed to migrate database: ", err)
	}
}

func main() {
	ConnectDatabase()

	router := fiber.New()

	router.Post("/signup", register)
	router.Post("/login", login)

	router.Get("/projects", jwtMiddleware, GetAllProjects)
	router.Get("/project/:id", jwtMiddleware, GetProject)
	router.Post("/project/add", jwtMiddleware, CreateProject)
	router.Post("/project/update/:id", jwtMiddleware, UpdateProject)
	router.Post("/project/delete/:id", jwtMiddleware, DeleteProject)

	router.Get("/tasks", jwtMiddleware, GetAllTasks)
	router.Get("/tasks/:id", jwtMiddleware, GetTaskById)
	router.Post("/tasks/add", jwtMiddleware, CreateTask)
	router.Post("/tasks/update/:id", jwtMiddleware, UpdateTask)
	router.Post("/tasks/delete/:id", jwtMiddleware, DeleteTask)
	router.Post("/tasks/assign/:id", jwtMiddleware, AssignTask)

	router.Get("/teams", jwtMiddleware, GetAllTeams)
	router.Post("/teams/add", jwtMiddleware, CreateTeam)
	router.Post("/teams/addmember", jwtMiddleware, AddTeamMember)
	router.Post("/teams/removemember/:id", jwtMiddleware, RemoveTeamMember)

	router.Get("/progress/projects", jwtMiddleware, ProgressProjects)
	router.Get("/progress/projects/:id", jwtMiddleware, ProgressProject)
	router.Get("/progress/tasks", jwtMiddleware, ProgressTasks)
	router.Get("/progress/tasks/:id", jwtMiddleware, ProgressTaskById)

	log.Fatal(router.Listen(":3000"))
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
	if err := DB.Where("user_name = ? AND password = ?", user.UserName, user.Password).First(&dbUser).Error; err != nil {
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

func CreateProject(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	project := new(Project)
	if err := c.BodyParser(project); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	project.Id = uuid.New().String()
	project.UserId = userID

	if err := DB.Create(project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create project"})
	}

	return c.Status(fiber.StatusCreated).JSON(project)
}

func GetAllProjects(c *fiber.Ctx) error {
	var projects []Project

	if err := DB.Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not retrieve projects"})
	}

	return c.JSON(projects)
}

func GetProject(c *fiber.Ctx) error {
	id := c.Params("id")
	var project Project
	if err := DB.First(&project, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	return c.JSON(project)
}

func UpdateProject(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("userID").(string)

	var project Project
	if err := DB.First(&project, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not the owner of this project"})
	}

	if err := c.BodyParser(&project); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	if err := DB.Save(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update project"})
	}

	return c.JSON(project)
}

func DeleteProject(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("userID").(string)

	if err := DB.Where("id = ? AND user_id = ?", id, userID).Delete(&Project{}).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not the owner of this project"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func CreateTask(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	task := new(Task)
	if err := c.BodyParser(task); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	task.Id = uuid.New().String()
	task.AssignedTo = userID

	if err := DB.Create(task).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create task"})
	}

	return c.Status(fiber.StatusCreated).JSON(task)
}

func GetAllTasks(c *fiber.Ctx) error {
	var tasks []Task

	if err := DB.Find(&tasks).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not retrieve tasks"})
	}

	return c.JSON(tasks)
}

func GetTaskById(c *fiber.Ctx) error {
	id := c.Params("id")
	var task Task
	if err := DB.First(&task, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
	}

	return c.JSON(task)
}

func UpdateTask(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("userID").(string)

	var task Task
	if err := DB.First(&task, "id = ? AND assigned_to = ?", id, userID).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not assigned to this task"})
	}

	if err := c.BodyParser(&task); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	if err := DB.Save(&task).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update task"})
	}

	return c.JSON(task)
}

func DeleteTask(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("userID").(string)

	if err := DB.Where("id = ? AND assigned_to = ?", id, userID).Delete(&Task{}).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not assigned to this task"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func AssignTask(c *fiber.Ctx) error {
	id := c.Params("id")
	//userID := c.Locals("userID").(string)

	var task Task
	if err := DB.First(&task, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
	}

	var input struct {
		AssignedTo string `json:"assigned_to"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	task.AssignedTo = input.AssignedTo
	if err := DB.Save(&task).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not assign task"})
	}

	return c.JSON(task)
}

func CreateTeam(c *fiber.Ctx) error {
	team := new(Team)
	if err := c.BodyParser(team); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	team.Id = uuid.New().String()

	if err := DB.Create(team).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create team"})
	}

	return c.Status(fiber.StatusCreated).JSON(team)
}

func GetAllTeams(c *fiber.Ctx) error {
	var teams []Team

	if err := DB.Find(&teams).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not retrieve teams"})
	}

	return c.JSON(teams)
}

func AddTeamMember(c *fiber.Ctx) error {
	member := new(TeamMember)
	if err := c.BodyParser(member); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	member.Id = uuid.New().String()

	if err := DB.Create(member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not add team member"})
	}

	return c.Status(fiber.StatusCreated).JSON(member)
}

func RemoveTeamMember(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := DB.Delete(&TeamMember{}, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not remove team member"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func ProgressProjects(c *fiber.Ctx) error {
	var projects []Project

	if err := DB.Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not retrieve projects"})
	}

	progress := make(map[string]float64)
	for _, project := range projects {
		var tasks []Task
		DB.Where("project_id = ?", project.Id).Find(&tasks)

		completed := 0
		for _, task := range tasks {
			if task.Status == "completed" {
				completed++ // ab ye dekh basic sa tha
			}
		}

		progress[project.Id] = float64(completed) / float64(len(tasks)) * 100
	}

	return c.JSON(progress)
}

func ProgressProject(c *fiber.Ctx) error {
	id := c.Params("id")
	var project Project
	if err := DB.First(&project, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var tasks []Task
	DB.Where("project_id = ?", project.Id).Find(&tasks)

	completed := 0
	for _, task := range tasks {
		if task.Status == "completed" {
			completed++
		}
	}

	progress := float64(completed) / float64(len(tasks)) * 100

	return c.JSON(fiber.Map{"project_id": project.Id, "progress": progress})
}

func ProgressTasks(c *fiber.Ctx) error {
	var tasks []Task

	if err := DB.Find(&tasks).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not retrieve tasks"})
	}

	progress := make(map[string]string)
	for _, task := range tasks {
		progress[task.Id] = task.Status
	}

	return c.JSON(progress)
}

func ProgressTaskById(c *fiber.Ctx) error {
	id := c.Params("id")
	var task Task
	if err := DB.First(&task, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
	}

	return c.JSON(fiber.Map{"task_id": task.Id, "status": task.Status})
}
