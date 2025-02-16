package students

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/Vladroon22/TG-Bot/internal/encryption"
)

type Student struct {
	groupName    string `json:"-"`
	Login        string `json:"login"`
	Password     string `json:"password"`
	Subscription bool   `json:"subscription"`
}

type Group struct {
	Relevance bool      `json:"relevance"`
	Users     []Student `json:"users"`
}

type GroupsData struct {
	Groups map[string]Group `json:"groups"`
}

var mutex sync.RWMutex

func readDataFromFile() (*GroupsData, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	data, err := os.ReadFile("./data.json")
	if err != nil {
		return nil, errors.New("Ошибка-открытия-json-файла")
	}

	var groupsData GroupsData
	if err := json.Unmarshal(data, &groupsData); err != nil {
		return nil, errors.New("Ошибка-декодирования-json")
	}

	if groupsData.Groups == nil {
		groupsData.Groups = make(map[string]Group)
	}

	return &groupsData, nil
}

func writeDataToFile(groupsData *GroupsData) error {
	mutex.Lock()
	defer mutex.Unlock()

	if groupsData.Groups == nil {
		groupsData.Groups = make(map[string]Group)
	}

	file, err := os.OpenFile("data.json", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.New("Ошибка-открытия-json")
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(groupsData); err != nil {
		return errors.New("Ошибка-декодирования-json")
	}

	return nil
}

func Register(c context.Context, groupName, login, password string) error {
	go func() error {
		select {
		case <-c.Done():
			return errors.New("введите данные заново")
		default:
			return nil
		}
	}()
	groupsData, err := readDataFromFile()
	if err != nil {
		return errors.New("Ошибка-открытия-json-при-регистрации")
	}

	student := Student{groupName: groupName, Login: login, Password: password, Subscription: false}
	group, ok := groupsData.Groups[groupName]
	if !ok {
		group = Group{Relevance: true, Users: []Student{student}}
	}
	for _, st := range group.Users {
		if strings.EqualFold(st.Login, login) {
			return errors.New("попробуйте другой логин")
		}
	}
	group.Users = append(group.Users, student)
	groupsData.Groups[groupName] = group

	if err := writeDataToFile(groupsData); err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func Enter(c context.Context, groupname, login, password string) (Student, error) {
	go func() error {
		select {
		case <-c.Done():
			return errors.New("введите данные заново")
		default:
			return nil
		}
	}()

	groupsData, err := readDataFromFile()
	if err != nil {
		return Student{}, errors.New("Ошибка-открытия-json-файла")
	}

	group, ok := groupsData.Groups[groupname]
	if !ok {
		return Student{}, errors.New("Группа-не-найдена")
	}

	for _, st := range group.Users {
		if strings.EqualFold(st.Login, login) {
			if encryption.CmpHashAndPass(st.Password, password) {
				return st, nil
			} else {
				return st, errors.New("Неверный-пароль")
			}
		}
	}

	return Student{}, errors.New("Студент-не-найден")
}

func (st *Student) ChangeStatus(status bool) error {
	groupsData, err := readDataFromFile()
	if err != nil {
		return errors.New("Ошибка-открытия-json-при-регистрации")
	}
	mutex.Lock()
	defer mutex.Unlock()

	group, ok := groupsData.Groups[st.groupName] // !!!!!
	if !ok {
		return errors.New("Группа-не-найдена")
	}

	found := false
	for i := range group.Users {
		if strings.EqualFold(group.Users[i].Login, st.Login) {
			group.Users[i].Subscription = status
			found = true
			break
		}
	}

	if !found {
		return errors.New("Студент-не-найден")
	}

	if err := writeDataToFile(groupsData); err != nil {
		return errors.New(err.Error())
	}

	return nil
}
