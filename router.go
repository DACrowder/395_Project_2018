// 395 Project Team Gold

package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

func logging(f http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		logger.Printf("'%v' %v", r.Method, r.URL.Path)
		f.ServeHTTP(w, r)

	})

}

func errorMessage(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errResp := context.Get(r, "error")
		if errResp != nil {
			fmt.Println("CONTEXT PASSED")
			w.WriteHeader(errResp.(int))
		}
		f.ServeHTTP(w, r)
	})
}

func checkSession(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "loginSession")
		if err != nil {
			//http.Error(w, err.Error(), http.StatusInternalServerError)
			session.Values["username"] = nil
			session.Save(r, w)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Retrieve our struct and type-assert it
		val := session.Values["username"]
		role := session.Values["role"]
		if val != nil {
			if strings.Contains(r.URL.Path, "/api/v1/admin") {
				if role == 3 {
					f.ServeHTTP(w, r)
				} else {
					logger.Println("Not authorized as admin")
					return
				}
			} else {
				logger.Println(val)
				f.ServeHTTP(w, r)
			}
		} else {
			if strings.Contains(r.URL.Path, "login") {
				f.ServeHTTP(w, r)
			} else {
				http.Redirect(w, r, "/login", http.StatusFound)
			}
		}
		return
	})
}
func createRouter() (*mux.Router, error) {
	r := mux.NewRouter()
	r.Use(logging)
	r.Use(errorMessage)

	r.StrictSlash(true)
	// static file handling (put assets in views folder)
	r.PathPrefix("/views/").Handler(http.StripPrefix("/views/", http.FileServer(http.Dir("./views/"))))

	// api routes+calls set up
	apiRoutes(r)

	r.HandleFunc("/", baseRoute)
	// setup admin routes
	adminRoutes(r)
	// load login pages html tmplts
	r.HandleFunc("/login", loadMainLogin)
	r.HandleFunc("/logout", handleLogout)
	l := r.PathPrefix("/login").Subrouter()
	l.HandleFunc("/facilitator", loadLogin)
	l.HandleFunc("/teacher", loadLogin)
	l.HandleFunc("/admin", loadLogin)

	r.Use(checkSession)

	//load dashboard and calendar pages
	r.HandleFunc("/dashboard", loadDashboard)
	r.HandleFunc("/calendar", loadCalendar)
	r.HandleFunc("/donate", loadDonate)
	r.HandleFunc("/change_password", loadPassword)

	return r, nil
}

func adminRoutes(r *mux.Router) {
	a := r.PathPrefix("/admin").Subrouter()
	a.HandleFunc("/dashboard", loadAdminDash)
	a.HandleFunc("/users", loadAdminUsers)
	a.HandleFunc("/reports", loadAdminReports)
	a.HandleFunc("/calendar", loadAdminCalendar)
	a.HandleFunc("/classes", loadAdminClasses)
}

func apiRoutes(r *mux.Router) {
	s := r.PathPrefix("/api/v1").Subrouter()
	s.HandleFunc("/admin/calendar/setup/", calSetup).Methods("POST")
	s.HandleFunc("/admin/calendar/setup/", undoSetup).Methods("DELETE")
	s.HandleFunc("/admin/users", getUserList).Methods("GET")
	s.HandleFunc("/admin/users", createUser).Methods("POST")
	s.HandleFunc("/admin/users", updateUser).Methods("PUT")
	s.HandleFunc("/admin/teachers", getTeachers).Methods("GET")
	s.HandleFunc("/admin/classes", getClassInfo).Methods("GET")
	s.HandleFunc("/admin/classes", createClass).Methods("POST")
	s.HandleFunc("/admin/classes", updateClass).Methods("PUT")
	s.HandleFunc("/admin/facilitators", lonelyFacilitators).Methods("GET")
	s.HandleFunc("/admin/families", getFamilyList).Methods("GET")
	s.HandleFunc("/admin/families", createFamily).Methods("POST")
	s.HandleFunc("/admin/families", updateFamily).Methods("GET")
	s.HandleFunc("/admin/dashboard", defaultReport).Methods("GET")
	s.HandleFunc("/dashboard", getDashData).Methods("GET")

	s.HandleFunc("/charts", monthlyReport).Methods("GET")

	/* Events JSON routes for scheduler system */
	s.HandleFunc("/events/{target}", getEvents).Methods("GET")
	s.HandleFunc("/events/{target}", eventPostHandler).Methods("POST")
	l := s.PathPrefix("/login").Subrouter()
	l.HandleFunc("/facilitator/", loginHandler).Methods("POST")
	l.HandleFunc("/teacher/", loginHandler).Methods("POST")
	l.HandleFunc("/admin/", loginHandler).Methods("POST")
}

//noinspection ALL
func baseRoute(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Base route to Caraway API, redirecting to main page")
	http.Redirect(w, r, "/login", http.StatusPermanentRedirect)
}

func loadDashboard(w http.ResponseWriter, r *http.Request) {
	pg, err := loadPage("dashboard", r) // load page
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	s := tmpls.Lookup("dashboard.tmpl")
	// dependency flags for dashboard
	pg.Calendar = true
	pg.Chart = true
	pg.Dashboard = true
	s.ExecuteTemplate(w, "dashboard", pg) // include page struct
}

func loadPassword(w http.ResponseWriter, r *http.Request) {
	pg, err := loadPage("password", r) // load page
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	s := tmpls.Lookup("password.tmpl")
	s.ExecuteTemplate(w, "password", pg) // include page struct
}

func loadDonate(w http.ResponseWriter, r *http.Request) {
	pg, err := loadPage("donate", r) // load page
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	s := tmpls.Lookup("donate.tmpl")
	s.ExecuteTemplate(w, "donate", pg) // include page struct
}

func loadCalendar(w http.ResponseWriter, r *http.Request) {
	pg, err := loadPage("calendar", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	s := tmpls.Lookup("calendar.tmpl")
	// calendar dependency flag
	pg.Calendar = true
	logger.Println(pg)
	logger.Println(s.ExecuteTemplate(w, "calendar", pg))
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "loginSession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Retrieve our struct and type-assert it
	session.Values["username"] = nil
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}
