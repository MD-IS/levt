package main

import (
  tea "github.com/charmbracelet/bubbletea"
  "path/filepath"
  "encoding/json"
  "strings"
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
  width       int
}

func (this Config) Init() tea.Cmd {
  return nil
}

func (this Config) Update(
  message tea.Msg,
) (tea.Model, tea.Cmd) {
  switch m := message.(type) {
    case tea.WindowSizeMsg: {
      this.width = m.Width
    }
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
              c := &this.Bookmarks
              if *i < len(*c) {
                *c = append((*c)[:*i], (*c)[*i+1:]...)
                this.Save()
                *i++
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
          if this.index < len(this.Bookmarks) {
            this.promptText = "Delete ? "
          }
        }
      }
    }
  }
  return this, nil
}

func (this Config) View() string {
  if this.exiting {return ""}
  str := "\x1b[1mBookmarks:\x1b[m\n"
  scs := append(this.Bookmarks, this.LastRead)
  for i, v := range scs {
    t := v.String()
    if len(t) + 2 > this.width {
      split := strings.Split(t, "]")
      last := len(split) - 1
      suffix := split[last]
      margin := 4 + len(suffix)
      if this.width > margin {
        rst := strings.Join(split[:last], "]")
        shrt := rst[:this.width - margin]
        t = shrt + "…]" + suffix
      }
    }

    if i == len(scs) -1 {
      str += "\x1b[1mLast Read:\x1b[m\n"
    }
    if this.index == i {
      str += "> \x1b[33m" + t + "\x1b[m\n"
    } else {
      str += "  " + t + "\n"
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
