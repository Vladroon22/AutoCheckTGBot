package students

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"

	"github.com/Vladroon22/TG-Bot/internal/utils"
)

type Student struct {
	id           int
	groupName    string
	Login        string `json:"login"`
	Password     string `json:"password"`
	Subscription bool   `json:"subscription"`
}

type Group struct {
	groupName string
	Relevance bool      `json:"relevance"`
	Users     []Student `json:"users"`
}

type GroupsData struct {
	Groups map[string]Group `json:"groups"`
}

var (
	cache map[string]Student
	mutex sync.RWMutex
)

func init() {
	cache = make(map[string]Student)
	mutex = sync.RWMutex{}
}

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

	file, err := os.OpenFile("data.json", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.New("open (write) json error")
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(groupsData); err != nil {
		return errors.New("encode json error")
	}

	return nil
}

func Register(groupName, login, password string) error {
	groupsData, err := readDataFromFile()
	if err != nil {
		return errors.New("open json for register error")
	}

	stud := []Student{}
	for _, group := range groupsData.Groups {
		stud = append(stud, group.Users...)
	}
	sort.Slice(stud, func(i, j int) bool { return stud[i].Login < stud[j].Login })
	i := sort.Search(len(stud), func(i int) bool { return stud[i].Login == login })
	if i >= len(stud) || stud[i].Login != login {
		return errors.New("попробуйте другой логин")
	}

	student := Student{Login: login, Password: password, Subscription: false}
	if group, ok := groupsData.Groups[groupName]; !ok {
		group = Group{Relevance: true, Users: []Student{student}}
		group.groupName = groupName
		groupsData.Groups[groupName] = group
		cache[groupName] = student
	} else {
		group.Users = append(group.Users, student)
		if _, ok := cache[groupName]; !ok {
			group.groupName = groupName
			cache[groupName] = student
		}
	}

	if err := writeDataToFile(groupsData); err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func Enter(groupname, login, password string) (Student, error) {
	groupsData, err := readDataFromFile()
	if err != nil {
		return Student{}, errors.New("open json error")
	}

	cacheStud, ok := cache[groupname]
	if ok {
		if !utils.CmpHashAndPass(cacheStud.Password, password) {
			return Student{}, errors.New("Неверный-пароль")
		}
		return cacheStud, nil
	}

	groups := []Group{}
	for name, group := range groupsData.Groups {
		group.groupName = name
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].groupName < groups[j].groupName })
	i := sort.Search(len(groups), func(i int) bool { return groups[i].groupName == groupname })
	if i >= len(groups) || groups[i].groupName != groupname {
		return Student{}, errors.New("Группа-не-найдена")
	}
	stud := groups[i].Users

	sort.Slice(stud, func(i, j int) bool { return stud[i].Login < stud[j].Login })
	idx := sort.Search(len(stud), func(i int) bool { return stud[i].Login == login })
	if idx >= len(stud) || stud[idx].Login != login {
		return Student{}, errors.New("Студент-не-найден")
	}
	findStudent := stud[idx]

	if !utils.CmpHashAndPass(findStudent.Password, password) {
		return Student{}, errors.New("Неверный-пароль")
	}

	if _, ok := cache[groupname]; !ok {
		cache[groupname] = stud[idx]
	}
	findStudent.groupName = groupname
	findStudent.id = idx

	return findStudent, nil
}

func (st *Student) ChangeStatus(status bool) error {
	groupsData, err := readDataFromFile()
	if err != nil {
		return errors.New("open json error")
	}

	group := groupsData.Groups[st.groupName]
	users := group.Users
	users[st.id].Subscription = status

	if err := writeDataToFile(groupsData); err != nil {
		return errors.New(err.Error())
	}

	return nil
}
