package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"github.com/nanobox-io/golang-scribble"
	"log"
	"net/http"
)

const REDIRECT_NUMBER = 302

type User struct {
	Username    string `form:"username"`
	Password    string `form:"password"`
	IsAdmin     bool
	IsAnonimous bool
}

type Comment struct {
	Username string
	Comment  string `form:"comment"`
}

type ShopItem struct {
	Name         string `form:"name"`
	Desc         string `form:"desc"`
	Id           string `form:"id"`
	Cost         int    `form:"cost"`
	Image        string `form:"image"`
	UserComments []Comment
}

func main() {
	gob.Register(User{})

	AnonimousUser := User{"Anonimuos", "", false, true}

	// init db
	db, err := scribble.New("./users", nil)

	itemDb, err := scribble.New("./items", nil)
	// init db end

	if err != nil {
		log.Println("Can not init db")
	}

	m := martini.Classic()

	m.Use(render.Renderer(render.Options{
		Layout:     "layout",
		Directory:  "templates",
		Extensions: []string{".tmpl", ".html"},
		Charset:    "UTF-8",
	}))

	m.Use(martini.Static("static"))
	store := sessions.NewCookieStore([]byte("secret123"))
	m.Use(sessions.Sessions("my_session", store))
	// sessions

	m.Get("/", func(r render.Render, s sessions.Session) {
		var context struct {
			User  User
			Items []ShopItem
		}
		var flag bool
		context.User, flag = s.Get("users").(User)
		if !flag {
			context.User = AnonimousUser
		}
		items, err := itemDb.ReadAll("items")
		if err != nil {
			log.Println("Can not read items from db")
		}
		shopItem := ShopItem{}
		tmpSl := make([]ShopItem, 0)
		for _, el := range items {
			err := json.Unmarshal([]byte(el), &shopItem)
			if err != nil {
				log.Println("Can not unmarshal element")
			}
			tmpSl = append(tmpSl, shopItem)
		}
		context.Items = tmpSl
		r.HTML(http.StatusOK, "index", context)
	})

	m.Get("/login", func(r render.Render) {
		r.HTML(http.StatusOK, "login", nil)
	})
	m.Post("/login", binding.Bind(User{}), func(r render.Render, u User, s sessions.Session) {
		fmt.Println(u)
		loginedUser := User{}
		err := db.Read("users", u.Username, &loginedUser)
		if err != nil {
			r.Text(http.StatusOK, "error")
		} else {
			s.Set("users", loginedUser)
			r.Redirect("/", REDIRECT_NUMBER)
		}
	})

	m.Get("/register", func(r render.Render) {
		r.HTML(http.StatusOK, "register", nil)
	})
	m.Post("/register", binding.Bind(User{}), func(r render.Render, u User) {
		u.IsAdmin = false
		err := db.Write("users", u.Username, u)
		if err != nil {
			r.Redirect("/register", http.StatusOK)
		} else {
			r.Redirect("/", REDIRECT_NUMBER)
		}
	})

	m.Get("/addnew", func(r render.Render) {
		r.HTML(http.StatusOK, "addnew", nil)
	})

	m.Post("/addnew", binding.Bind(ShopItem{}), func(r render.Render, item ShopItem) {
		itemDb.Write("items", item.Id, item)
		r.Redirect("/", REDIRECT_NUMBER)
	})

	m.Get("/showitem/:id", func(r render.Render, p martini.Params, s sessions.Session) {
		id := p["id"]
		item := ShopItem{}
		err := itemDb.Read("items", id, &item)
		if err != nil {
			r.Redirect("/", REDIRECT_NUMBER)
		}
		var context struct {
			User User
			Item ShopItem
		}
		context.Item = item
		var flag bool
		context.User, flag = s.Get("users").(User)
		if !flag {
			context.User = AnonimousUser
		}
		r.HTML(http.StatusOK, "showitem", context)
	})

	m.Post("/showitem/:id", binding.Bind(Comment{}), func(c Comment, s sessions.Session, r render.Render, p martini.Params) {
		item := ShopItem{}
		user, flag := s.Get("users").(User)
		if !flag {
			r.Redirect("/login")
		}
		c.Username = user.Username
		err := itemDb.Read("items", p["id"], &item)
		slice := item.UserComments[:]
		slice = append(slice, c)
		item.UserComments = slice
		fmt.Println(item)
		if err != nil {
			log.Println("can not read object from db items")
		}
		err = itemDb.Write("items", item.Id, item)
		if err != nil {
			log.Println("Can not write to db")
		}
		r.Redirect("/showitem/" + p["id"])
	})

	m.Get("/logout", func(r render.Render, s sessions.Session) {
		s.Delete("users")
		r.Redirect("/login", REDIRECT_NUMBER)
	})

	m.Run()

}
