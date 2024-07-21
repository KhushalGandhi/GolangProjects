package main

//
//import (
//	"github.com/gofiber/fiber/v2"
//	"github.com/golang-jwt/jwt/v4"
//	"github.com/google/uuid"
//	"gorm.io/driver/postgres"
//	"gorm.io/gorm"
//	"log"
//	"time"
//)
//
//type User struct {
//	ID       string `gorm:"primaryKey"  json:"id"`
//	UserName string `gorm:"unique" json:"user_name"`
//	Password string `json:"password"`
//}
//
//type Project struct {
//	Id          string `json:"id"`
//	Description string `json:"description"`
//	Name        string `json:"name"`
//	StartDate   string `json:"start_date"`
//	EndDate     string `json:"end_date"`
//	UserId      string `json:"user_id"` // who created the project
//}
//
//type Task struct {
//	Id          string `json:"id"`
//	Name        string `json:"name"`
//	Description string `json:"description"`
//	Status      string `json:"status"`
//	DueDate     string `json:"due_date"`
//	ProjectId   string `json:"project_id"`
//	AssignedTo  string `json:"assigned_to"`
//}
//
//type Team struct {
//	Id   string `json:"id"`
//	Name string `json:"name"`
//	// isko marhsl unmarshal se bhi kr skte hai but not feasible kyunki baar baar nya member aayega team mein to dikkt dega
//
//} // not sure inko 2 mein kyun batan
//
//type TeamMember struct {
//	Id     string `json:"id"`
//	TeamId string `json:"team_id"`
//	UserId string `json:"user_id"`
//}
//
//var jwtSecret = []byte("secret")
//
//var DB *gorm.DB
//
//func ConnectDatabase() {
//	dsn := "host=localhost user=postgres password=new+password dbname=social_media port=5432 sslmode=disable TimeZone=Asia/Shanghai"
//	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
//	if err != nil {
//		log.Fatal("failed to connect to database: ", err)
//	}
//
//	DB = database
//
//	err = DB.AutoMigrate(&User{}, &Team{}, &TeamMember{}, &Task{}, &Project{})
//	if err != nil {
//		log.Fatal("failed to migrate database: ", err)
//	}
//}
//
//func main() {
//
//	ConnectDatabase()
//
//	router := fiber.New()
//
//	router.Post("/signUp", register)
//	router.Post("/login", login)
//
//	router.Get("/projects", GetAllProjects)
//	router.Get("/project/:id", GetProject)
//	router.Post("/project/add", CreateProject)
//	router.Post("/projects/update/:id")
//	router.Post("/projects/delete/:id")
//
//	router.Get("/tasks")
//	router.Get("/tasks/:id")
//	router.Post("/tasks/add")
//	router.Post("/tasks/update/:id")
//	router.Post("/tasks/delete/:id")
//	router.Post("/tasks/assign/:id")
//
//	router.Get("/teams")
//
//	log.Fatal(router.Listen(":3000"))
//
//}
//
//func jwtMiddleware(c *fiber.Ctx) error {
//	authHeader := c.Get("Authorization")
//	if authHeader == "" {
//		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or malformed JWT"})
//	}
//
//	tokenStr := authHeader[len("Bearer "):]
//	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
//		return jwtSecret, nil
//	})
//
//	if err != nil || !token.Valid {
//		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired JWT"})
//	}
//
//	claims, ok := token.Claims.(jwt.MapClaims)
//	if !ok {
//		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid JWT claims"})
//	}
//
//	c.Locals("userID", claims["userID"])
//	return c.Next()
//}
//
//func register(c *fiber.Ctx) error {
//	user := new(User)
//	if err := c.BodyParser(user); err != nil {
//		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
//	}
//
//	user.ID = uuid.New().String()
//
//	if err := DB.Create(user).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not register user"})
//	}
//
//	return c.Status(fiber.StatusCreated).JSON(user)
//}
//
//func login(c *fiber.Ctx) error {
//	user := new(User)
//	if err := c.BodyParser(user); err != nil {
//		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
//	}
//
//	var dbUser User
//	if err := DB.Where("username = ? AND password = ?", user.UserName, user.Password).First(&dbUser).Error; err != nil {
//		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
//	}
//
//	token := jwt.New(jwt.SigningMethodHS256)
//	claims := token.Claims.(jwt.MapClaims)
//	claims["userID"] = dbUser.ID
//	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()
//
//	t, err := token.SignedString(jwtSecret)
//	if err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not login"})
//	}
//
//	return c.JSON(fiber.Map{"token": t})
//}
//
//func CreateProject(c *fiber.Ctx) error {
//	username := c.Locals("user_name").(string) // ye type conversion hai na yaad rkh
//
//	project := new(Project)
//
//	project.Id = uuid.New().String()
//	project.UserId = username
//
//	if err := DB.Create(project).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create project"})
//	}
//
//	return c.Status(fiber.StatusCreated).JSON(project)
//
//}
//
//func GetAllProjects(c *fiber.Ctx) error {
//	var project []Project
//
//	if err := DB.Preload("User").Find(&project).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not retrieve projects"})
//	}
//
//	return c.JSON(project)
//}
//
//func GetProject(c *fiber.Ctx) error {
//	id := c.Params("id")
//	var project Project
//	if err := DB.Preload("User").First(&project, "id = ?", id).Error; err != nil {
//		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
//	}
//
//	return c.JSON(project)
//
//}
//
//func UpdateProject(c *fiber.Ctx) error {
//	id := c.Params("id")
//	userID := c.Locals("userID").(string) // iska concept bhi theek krna hai
//
//	var project Project
//	if err := DB.First(&project, "id = ? AND user_id = ?", id, userID).Error; err != nil {
//		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not the owner of this project"})
//	}
//	// ab ye trh ka edge case agr ye hi nhi kr paayega tb normal coding nhi dhund paayega
//
//	// similar to here more the edge cases get difficult more the coding u will require
//
//	if err := c.BodyParser(&project); err != nil {
//		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
//	}
//
//	if err := DB.Save(&project).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update project"})
//	}
//
//	return c.JSON(project)
//}
//
//// my handling of the pain is at the lowest there have been at the moment in my life
//// i need to build like it any other muscle like i do in gym
//
//func DeleteProject(c *fiber.Ctx) error {
//	id := c.Params("id")                  // normal hai ki key hmein milegi jo hmein delete krni hai projectId
//	userID := c.Locals("userID").(string) // and ye jo userId aayegi token ke through
//
//	if err := DB.Where("id = ? AND user_id = ?", id, userID).Delete(&Project{}).Error; err != nil {
//		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not the owner of this project "})
//	}
//
//	return c.SendStatus(fiber.StatusNoContent)
//}
//
//func CreateTask(c *fiber.Ctx) error {
//	projectID := c.Locals("project_id").(string)
//
//	task := new(Task)
//	if err := c.BodyParser(task); err != nil {
//		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
//	}
//
//	task.Id = uuid.New().String()
//	task.ProjectId = projectID
//
//	if err := DB.Create(task).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create task"})
//	}
//
//	return c.Status(fiber.StatusCreated).JSON(task)
//}
//
//func GetAllTasks(c *fiber.Ctx) error {
//	var tasks []Task
//
//	if err := DB.Preload("User").Find(&tasks).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not retrieve tasks"})
//	}
//
//	return c.JSON(tasks)
//
//}
//
//func GetTaskById(c *fiber.Ctx) error {
//	id := c.Params("id")
//
//	var task Task
//	if err := DB.Preload("User").First(&task, "id = ?", id).Error; err != nil {
//		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
//	}
//
//	return c.JSON(task)
//
//}
//
//func UpdateATask(c *fiber.Ctx) error {
//	id := c.Params("id")
//	userID := c.Locals("userID").(string)
//
//	var task Task
//	if err := DB.First(&task, "id = ? AND user_id = ?", id, userID).Error; err != nil {
//		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not the owner of this task"})
//	}
//
//	if err := c.BodyParser(&task); err != nil {
//		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
//	}
//
//	if err := DB.Save(&task).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update task"})
//	}
//
//	return c.JSON(task)
//}
//
//func DeleteATask(c *fiber.Ctx) error {
//	return c.Status(fiber.StatusOK).JSON("threat")
//}
//
//func GetAllTeams(c *fiber.Ctx) error {
//	var team []Team
//	if err := DB.Preload("User").Find(&team).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not retrieve posts"})
//	}
//
//	return c.JSON(team)
//
//}
//
//func CreateTeam(c *fiber.Ctx) error {
//	userID := c.Locals("user_id").(string)
//
//	team := new(Team)
//	if err := c.BodyParser(team); err != nil {
//		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
//	}
//
//	team.Id = uuid.New().String()
//
//	if err := DB.Create(team).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create team"})
//	}
//
//	teamMember := new(TeamMember)
//
//	teamMember.TeamId = team.Id
//	teamMember.Id = uuid.New().String()
//	teamMember.UserId = userID // hopefully this logic is correct
//
//	// now the question is can we add a list of teammembers together in one go
//
//	if err := DB.Create(teamMember).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create team"})
//	}
//
//	return c.Status(fiber.StatusCreated).JSON(team)
//
//}
//
//func AddTeamMembers(c *fiber.Ctx) error {
//	teamId := c.Params("team_id")
//	userId := c.Locals("user_id").(string)
//
//	var teamMember TeamMember
//
//	teamMember.TeamId = teamId
//	teamMember.UserId = userId
//
//	if err := DB.Create(teamMember).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create team"})
//	}
//
//	return c.Status(fiber.StatusCreated).JSON(teamMember)
//
//}
//
//func RemoveTeamMember(c *fiber.Ctx) error {
//	teamMemberId := c.Params("id")
//
//	userID := c.Locals("userID").(string)
//
//	if err := DB.Where("id = ? AND user_id = ?", teamMemberId, userID).Delete(&TeamMember{}).Error; err != nil {
//		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you cannot delete this member"})
//	}
//
//	return c.SendStatus(fiber.StatusNoContent)
//
//}
//
//func ProgressTasks(c *fiber.Ctx) error {
//	var task []Task
//
//	if err := DB.Preload("User").Find(&task).Error; err != nil {
//		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not retrieve Task Status"})
//	}
//
//	return c.JSON(task)
//}
//
//func ProgressTaskId(c *fiber.Ctx) error {
//	id := c.Params("id")
//
//	var post Task
//	if err := DB.Preload("User").First(&post, "id = ?", id).Error; err != nil {
//		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
//	}
//
//	return c.JSON(post)
//
//}
