package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type APIServer struct {
	listenAdrr string
	store      Storage
}

func newAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAdrr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	r := mux.NewRouter()

	r.HandleFunc("/login", makeHTTPHandleFunc(s.handleLogin))
	r.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	r.HandleFunc("/account/{id}", WithJWTAuth(makeHTTPHandleFunc(s.getAccountbyID), s.store))
	r.HandleFunc("/delete-account/{id}", makeHTTPHandleFunc(s.deleteAccount))
	r.HandleFunc("/transfer", makeHTTPHandleFunc(s.handleTransfer))
	log.Println("Server run on port: ", s.listenAdrr)
	http.ListenAndServe(s.listenAdrr, r)
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method not allowed %s", r.Method)
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	acc, err := s.store.GetAccountByNumber(int(req.Number))
	if err != nil {
		return err
	}

	if !acc.ValidPassword(req.Password) {
		return fmt.Errorf("not authenticated")
	}

	token, err := createJWT(acc)
	if err != nil {
		return err
	}

	resp := LoginResponse{
		Token:  token,
		Number: acc.Number,
	}

	return WriteJSON(w, http.StatusOK, resp)
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.getAccounts(w, r)
	}
	if r.Method == "POST" {
		return s.createAccount(w, r)
	}
	return fmt.Errorf("method not allowed %s", r.Method)
}
func (s *APIServer) getAccounts(w http.ResponseWriter, _ *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, accounts)
}
func (s *APIServer) getAccountbyID(w http.ResponseWriter, r *http.Request) error {
	id, err := GetID(r)
	if err != nil {
		return err
	}
	account, err := s.store.GetAccountByID(id)
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, account)
}
func (s *APIServer) createAccount(w http.ResponseWriter, r *http.Request) error {
	accountReq := new(CreatedAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(accountReq); err != nil {
		return err
	}
	account, err := newAccount(accountReq.FirstName, accountReq.LastName, accountReq.Password)
	if err != nil {
		return err
	}

	if err := s.store.CreateAccount(account); err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, account)
}
func (s *APIServer) deleteAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "DELETE" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return fmt.Errorf("method not allowed")
	}
	id, err := GetID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return err
	}
	if err := s.store.DeleteAccount(id); err != nil {
		if err.Error() == fmt.Sprintf("no account found with id %d", id) {
			http.Error(w, "account not found", http.StatusNotFound)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return err
	}
	msg := fmt.Sprintf("id (%d) has been deleted", id)

	return WriteJSON(w, http.StatusOK, msg)
}
func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	transferReq := new(TransferRequest)
	if err := json.NewDecoder(r.Body).Decode(transferReq); err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, transferReq)
}
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func permissionDenied(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden, APIError{Error: "permission denied"})
}

func createJWT(account *Account) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt":     15000,
		"accountNumber": account.Number,
	}
	secret := os.Getenv("JWT_TOKEN")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func WithJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("calling JWT Auth")

		// Extract the JWT from the request header
		tokenString := r.Header.Get("x-jwt-token")

		// Validate the JWT
		token, err := validateJWT(tokenString)
		if err != nil || !token.Valid {
			permissionDenied(w)
			return
		}

		// Retrieve the user ID from the request
		userID, err := GetID(r)
		if err != nil {
			permissionDenied(w)
			return
		}

		// Get the account details associated with the user ID
		account, err := s.GetAccountByID(userID)
		if err != nil {
			permissionDenied(w)
			return
		}

		// Extract claims from the token and compare account number
		claims := token.Claims.(jwt.MapClaims)
		if account.Number != int64(claims["accountNumber"].(float64)) {
			permissionDenied(w)
			return
		}
		handlerFunc(w, r)
	}
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_TOKEN")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type APIError struct {
	Error string `json:"error"`
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		}
	}
}

func GetID(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return id, fmt.Errorf("invalid given id %s", idStr)
	}
	return id, nil
}
