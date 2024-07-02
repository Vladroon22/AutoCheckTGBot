package students

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/Vladroon22/TG-Bot/internal/encryption"
)

type Student struct {
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

var filemutex sync.RWMutex

func readDataFromFile() (*GroupsData, error) {
	filemutex.RLock()
	defer filemutex.RUnlock()

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
	filemutex.Lock()
	defer filemutex.Unlock()

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

func Register(groupName, login, password string) error {
	groupsData, err := readDataFromFile()
	if err != nil {
		return errors.New("Ошибка-открытия-json-при-регистрации")
	}

	student := Student{Login: login, Password: password, Subscription: false}

	group, ok := groupsData.Groups[groupName]
	if !ok {
		group = Group{Relevance: true, Users: []Student{student}}
	} else {
		group.Users = append(group.Users, student)
	}

	groupsData.Groups[groupName] = group

	return writeDataToFile(groupsData)
}

func Enter(groupname, login, password string) (*Student, error) {
	groupsData, err := readDataFromFile()
	if err != nil {
		return nil, errors.New("Ошибка-открытия-json-файла")
	}
	for name, group := range groupsData.Groups {
		if strings.EqualFold(name, groupname) {
			for _, st := range group.Users {
				if strings.EqualFold(st.Login, login) && encryption.CmpHashAndPass(st.Password, password) {
					return &st, nil
				}
			}
			return nil, errors.New("Студент-не-найден")
		}
	}
	return nil, errors.New("Группа-не-найдена")
}

func (st *Student) ChangeStatus(status bool) error {
	groupsData, err := readDataFromFile()
	if err != nil {
		return errors.New("Ошибка-открытия-json-при-регистрации")
	}

	if group, ok := groupsData.Groups[st.Login]; ok {
		for i := range group.Users {
			if strings.EqualFold(group.Users[i].Login, st.Login) {
				group.Users[i].Subscription = status
				break
			}
		}
	}
	return writeDataToFile(groupsData)
}
