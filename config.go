package main

import (
  tea "github.com/charmbracelet/bubbletea"
  "path/filepath"
  "encoding/json"
  "fmt"
  "os"
)

const CONF_DIRNAME = "levt"

type StartConf struct {
  Title     string  `json:"title"`
  Index     int     `json:"index"`
  Cursor    int     `json:"cursor"`
  IsXHTML   bool    `json:"isxhtml"`
  FilePath  string  `json:"filePath"`
}

type Config struct {
  LastRead    StartConf `json:"lastRead"`
  Bookmarks []StartConf `json:"bookmarks"`

  onExit      func(int)
  promptText  string
  prompt      bool
  exiting     bool
  index       int
}

func (this Config) Init() tea.Cmd {
  return nil
}

func (this Config) Update(
  message tea.Msg,
) (tea.Model, tea.Cmd) {
  switch m := message.(type) {
    case tea.KeyMsg: {
      i := &this.index
      switch m.String() {
        case "q": {
          if this.promptText != "" {
            this.promptText = ""
            this.prompt = false
          } else {
            this.onExit(-1)
            this.exiting = true
            return this, tea.Quit
          }
        }
        case "enter": {
          if this.promptText != "" {
            if this.prompt {
              if *i != 0 {
                c := &this.Bookmarks
                *c = append((*c)[:*i-1], (*c)[*i:]...)
                this.Save()
                *i--
              }
            }
            this.promptText = ""
            this.prompt = false
          } else {
            this.onExit(this.index)
            this.exiting = true
            return this, tea.Quit
          }
        }
        case "up", "left": {
          if this.promptText != "" {
            this.prompt = !this.prompt
          } else {
            if *i > 0 {
              *i--
            } else {
              *i = len(this.Bookmarks)
            }
          }
        }
        case "down", "right": {
          if this.promptText != "" {
            this.prompt = !this.prompt
          } else {
            if *i < len(this.Bookmarks) {
              *i++
            } else {
              *i = 0
            }
          }
        }
        case "d": {
          if this.index == 0 {return this, nil}
          this.promptText = "Delete ? "
        }
      }
    }
  }
  return this, nil
}

func (this Config) View() string {
  if this.exiting {return ""}
  str := "\x1b[1mLast Read:\x1b[m\n"
  lrs := this.LastRead.String()
  if this.index == 0 {
    str += "> \x1b[33m" + lrs + "\x1b[m\n"
  } else {
    str += "  " + lrs + "\n"
  }
  str += "\x1b[1mBookmarks:\x1b[m\n"
  for i, v := range this.Bookmarks {
    if this.index == (i + 1) {
      str += "> \x1b[33m" + v.String() + "\x1b[m\n"
    } else {
      str += "  " + v.String() + "\n"
    }
  }
  if this.promptText != "" {
    str += this.promptText
    if this.prompt {
      str += "\x1b[1;41m Yes \x1b[m"
    } else {
      str += " Yes "
    }
    if !this.prompt {
      str += "\x1b[1;44m No \x1b[m"
    } else {
      str += " No "
    }
  }
  return str
}

func (this *Config) StartConfig() error {
  return tea.NewProgram(this).Start()
}

func (this *Config) Save() error {
  byt, err := json.Marshal(this)
  if err != nil {return err}

  path := getConfigPath()
  dir := filepath.Dir(path)
  err = os.MkdirAll(dir, 0700)
  if err != nil {return err}

  err = os.WriteFile(path, byt, 0666)
  if err != nil {return err}

  return nil
}

func (this *StartConf) String() string {
  return fmt.Sprintf(
    "[%s]:%d:%d",
    this.Title,
    this.Index + 1,
    this.Cursor,
  )
}

func getConfig() (config Config, err error) {
  path := getConfigPath()
  byt, err := os.ReadFile(path)
  if err != nil {byt = []byte("{}")}
  err = json.Unmarshal(byt, &config)
  return
}

func getConfigPath() string {
  configDir, _ := os.UserConfigDir()
  ps := string(os.PathSeparator)

  configPath := fmt.Sprintf(
    "%s%sconfig.json",
    configDir + ps,
    CONF_DIRNAME + ps,
  )
  return configPath
}
