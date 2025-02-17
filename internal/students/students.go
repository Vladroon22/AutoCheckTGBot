package students

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"

	"github.com/Vladroon22/TG-Bot/internal/utils"
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
		return nil, errors.New("open json error")
	}

	var groupsData GroupsData
	if err := json.Unmarshal(data, &groupsData); err != nil {
		return nil, errors.New("unmarshal error json")
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
		return errors.New("open json error")
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(groupsData); err != nil {
		return errors.New("encode json error")
	}

	return nil
}

func Register(c context.Context, groupName, login, password string) error {
	groupsData, err := readDataFromFile()
	if err != nil {
		return errors.New("open json for register error")
	}

	student := Student{groupName: groupName, Login: login, Password: password, Subscription: false}
	group, ok := groupsData.Groups[groupName]
	if !ok {
		return errors.New("Групппа-не-найдена")
	}

	studs := group.Users
	sameLogin := func(st []Student) string {
		sort.Slice(st, func(i, j int) bool { return st[i].Login < st[j].Login })
		i := sort.Search(len(st), func(i int) bool { return st[i].Login == login })
		if i < len(st) && st[i].Login == login {
			return st[i].Login
		}
		return ""
	}(studs)

	if sameLogin != "" {
		return errors.New("попробуйте другой логин")
	}
	group.Users = append(group.Users, student)

	if groupsData.Groups == nil {
		groupsData.Groups = make(map[string]Group)
	}
	groupsData.Groups[groupName] = group

	if err := writeDataToFile(groupsData); err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func Enter(c context.Context, groupname, login, password string) (Student, error) {
	groupsData, err := readDataFromFile()
	if err != nil {
		return Student{}, errors.New("open json error")
	}

	group, ok := groupsData.Groups[groupname]
	if !ok {
		return Student{}, errors.New("Группа-не-найдена")
	}

	studs := group.Users
	st, err := func(st []Student) (Student, error) {
		sort.Slice(st, func(i, j int) bool { return st[i].Login < st[j].Login })
		i := sort.Search(len(st), func(i int) bool { return st[i].Login == login })
		if i < len(st) && st[i].Login == login {
			if !utils.CmpHashAndPass(st[i].Password, password) {
				return Student{}, errors.New("Неверный-пароль")
			} else {
				return st[i], nil
			}
		}
		return Student{}, errors.New("Студент-не-найден")
	}(studs)
	return st, err
}

func (st *Student) ChangeStatus(status bool) error {
	groupsData, err := readDataFromFile()
	if err != nil {
		return errors.New("open json error")
	}
	mutex.Lock()
	defer mutex.Unlock()

	group, ok := groupsData.Groups[st.groupName]
	if !ok {
		return errors.New("Группа-не-найдена")
	}

	studs := group.Users
	i, err := func(stud []Student) (int, error) {
		sort.Slice(stud, func(i, j int) bool { return stud[i].Login < stud[j].Login })
		i := sort.Search(len(stud), func(i int) bool { return stud[i].Login == st.Login })
		if i < len(stud) && stud[i].Login == st.Login {
			return i, nil
		}
		return 0, errors.New("Студент-не-найден")
	}(studs)

	if err != nil {
		return errors.New(err.Error())
	}

	studs[i].Subscription = status

	if err := writeDataToFile(groupsData); err != nil {
		return errors.New(err.Error())
	}

	return nil
}
