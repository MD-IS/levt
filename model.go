package main

import (
  "os"
  "io"
  "fmt"
  "time"
  "regexp"
  "strings"
  "path/filepath"

  tea "github.com/charmbracelet/bubbletea"
)

var linkRe *regexp.Regexp = regexp.MustCompile(
  `##link:([^;]+);`,
)

type EpubViewer struct {
  DebugMode       bool
  EPUBTitle       string
  FilePath        string
  Pages     [][][]string
  Logs          []string
  Hint            string

  Page            int
  Index           int
  Width           int
  Height          int
  Cursor          int
  PageLen       []int
  EpubItems     []EpubItem
  Hyperlinks  map[int]string
}

func (this *EpubViewer) RenderText(cursor int) {
  t := time.Now()

  this.Hyperlinks = make(map[int]string)
  this.Pages = [][][]string{}

  this.SetPages(cursor + this.Height)

  this.Cursor = cursor
  this.Page = 0
  last := len(this.Pages) - 1
  for i, v := range this.PageLen {
    if this.Cursor >= v {
      this.Page = i + 1
      if this.Page > last {
        this.Page = last
      }
    }
  }

  this.Logs = append(this.Logs, fmt.Sprintf(
    "%f: RenderText %d size %dx%d",
    time.Since(t).Seconds(),
    this.Index,
    this.Width,
    this.Height,
  ))
}

func (this EpubViewer) Init() tea.Cmd {
  return nil
}

func (this EpubViewer) Update(
  message tea.Msg,
) (tea.Model, tea.Cmd) {
  switch msg := message.(type) {
    case tea.WindowSizeMsg: {
      this.Height = msg.Height - 1
      this.Width = msg.Width - 4

      item := this.EpubItems[this.Index]
      if item.Close != nil {item.Close()}

      this.RenderText(this.Cursor)
    }
    case tea.KeyMsg: {
      key := msg.String()
      if this.DebugMode {return this.debug(key), nil}

      c := &this.Cursor
      p := &this.Page

      pl := this.PageLen
      end := 0
      if len(pl) > 0 {end = pl[len(pl) - 1]}

      this.Hint = ""
      switch key {
        case "enter": {
          link, ok := this.Hyperlinks[*c]
          if ok {
            base := this.EpubItems[this.Index].Href
            lastSlash := strings.LastIndex(base, "/")
            absoluteRe := regexp.MustCompile(
              `^([^:/]+:/)?/`,
            )
            upDirRe := regexp.MustCompile(
              `[^/]+/\.\./`,
            )

            if !absoluteRe.MatchString(link) {
              absolute := base[:lastSlash + 1] + link
              link = upDirRe.ReplaceAllString(
                absolute, "",
              )
            }

            this.Logs = append(
              this.Logs, "Enter :" + link,
            )

            var rerender bool
            for i, item := range this.EpubItems {
              if item.Href == link {
                rerender = true
                this.Index = i
              }
            }

            if rerender {
              item := this.EpubItems[this.Index]
              if item.Close != nil {item.Close()}

              this.RenderText(0)
            } else {
              err := open(link)
              if err != nil {
                this.Hint = "\x1b[41m Cannot open " +
                  link + " \x1b[m"
              }
            }
          }
        }
        case "ctrl+a", "home": {
          *c = 0
          *p = 0
        }
        case "ctrl+e", "end": {
          *c = this.SetPages(-1)
          *p = len(this.Pages) - 1
        }

        case "k", "up": {
          if *c > 0 {*c--}
          if *p > 0 && pl[*p - 1] > *c {
            *c = pl[*p - 1] - 1
            *p--
          }
        }
        case "j", "down": {
          if *c < end {*c++}
          if *p < (len(pl) - 1) && pl[*p] <= *c {
            *c = pl[*p]
            *p++
          }
        }
        case "q", "ctrl+d": {
          if this.EPUBTitle == "" {
            this.EPUBTitle = this.FilePath
          }

          config, _ := getConfig()
          fp, _ := filepath.Abs(this.FilePath)
          config.LastRead = StartConf{
            IsXHTML: this.EPUBTitle == this.FilePath,
            Title: this.EPUBTitle,
            Index: this.Index,
            FilePath: fp,
            Cursor: *c,
          }
          config.Save()
          return this, tea.Quit
        }
        case "ctrl+b": {
          if this.EPUBTitle == "" {
            this.EPUBTitle = this.FilePath
          }

          config, _ := getConfig()
          fp, _ := filepath.Abs(this.FilePath)
          b := StartConf{
            IsXHTML: this.EPUBTitle == this.FilePath,
            Title: this.EPUBTitle,
            Index: this.Index,
            FilePath: fp,
            Cursor: *c,
          }
          config.Bookmarks = append(
            config.Bookmarks, b,
          )
          if err := config.Save(); err == nil {
            h := fmt.Sprintf(
              "\x1b[44m Bookmark [+] %s \x1b[m",
              b.String(),
            )
            this.Logs = append(this.Logs, h)
            this.Hint = h
          } else {
            this.Hint = "\x1b[41m unable to save Bookmark \x1b[m"
          }
        }

        case "d": {
          this.DebugMode = true
        }

        case " ": {
          *c = pl[*p]
          if *p < (len(pl) - 1) {*p++}
        }
        case "backspace": {
          if *p > 1 {
            *c = pl[*p - 2]
            *p--
          } else {
            *c = 0
            *p = 0
          }
        }

        case "left": {
          if this.Index > 0 {
            item := this.EpubItems[this.Index]
            if item.Close != nil {item.Close()}

            this.Index--
            this.RenderText(0)
          }
        }
        case "right": {
          if this.Index + 1 < len(this.EpubItems) {
            item := this.EpubItems[this.Index]
            if item.Close != nil {item.Close()}

            this.Index++
            this.RenderText(0)
          }
        }
      }
    }
  }

  last := len(this.Pages) - 1
  if this.Page == last {this.SetPages(this.Height)}

  return this, nil
}

func (this *EpubViewer) debug(key string) tea.Model {
  switch key {
    case "q", "d": {this.DebugMode = false}

    case "ctrl+l": {
      this.Logs = []string{}
    }

    case "ctrl+r": {
      item := this.EpubItems[this.Index]
      if item.Close != nil {item.Close()}

      this.Page = 0
      this.Cursor = 0
      this.RenderText(0)
    }

    case "ctrl+d": {
      byt := []byte(fmt.Sprintf("%+v\n", this))
      os.WriteFile("dumpfile", byt, 0666)
    }
  }

  return this
}

func (this *EpubViewer) SetPages(max int) int {
  index := this.Index
  if index < 0 || index >= len(this.EpubItems) {
    return -1
  }

  raw := &this.EpubItems[index]
  o := raw.Offset
  if raw.Decoder == nil {raw.Load()}

  var clen int
  var c [][]string
  w := this.Width
  if len(this.Pages) != 0 {
    l := len(this.Pages) - 1

    c = this.Pages[l]
    for _, v := range c {
      clen += len(v)
    }

    this.Pages = this.Pages[:l]
    this.PageLen = this.PageLen[:l]
  }

  count := 0
  for max == -1 || count < max {
    err := raw.Line()
    if err == io.EOF {break}
    count++
  }

  for i := o; i < (o + count); i++ {
    line := raw.Content[i]
    if linkRe.MatchString(line) {
      m := linkRe.FindStringSubmatch(line)
      line = linkRe.ReplaceAllString(line, "")
      this.Hyperlinks[i] = strings.Split(m[1], "#")[0]
    }

    var p []string
    for _, v := range WordWrap(line, w) {
      if this.Height <= clen {
        newPage := append(c, p)
        this.Pages = append(this.Pages, newPage)
        c = [][]string{}
        p = []string{}
        clen = 0
      }

      p = append(p, v)
      clen++
    }
    c = append(c, p)
  }
  if len(c) > 0 {this.Pages = append(this.Pages, c)}

  var vlen int
  this.PageLen = make([]int, len(this.Pages))
  for i, v := range this.Pages {
    l := len(v) - 1
    this.PageLen[i] = vlen + l
    vlen += l
  }

  return vlen
}

func (this EpubViewer) View() string {
  index := this.Index
  items := this.EpubItems
  item := items[index]
  hint := fmt.Sprintf(
    "\x1b[7m %d/%d %s \x1b[m",
    index+1, len(items), item.Href,
  )

  if !this.DebugMode {
    p := this.Page
    arr := this.Pages
    if len(arr) <= p {return ""}

    var vlen int
    if p > 0 {vlen = this.PageLen[p - 1]}

    var c []string
    for i, v := range arr[p] {
      prefix := "\x1b[m  "
      if this.Cursor == (vlen + i) {
        prefix = "\x1b[7m \x1b[m "
        if link, ok := this.Hyperlinks[vlen + i]; ok {
          hint = fmt.Sprintf(
            "\x1b[7m Press ENTER to open %s \x1b[m",
            link,
          )
        }
      }

      for _, j := range v {c = append(c, prefix + j)}
    }

    for len(c) < this.Height {c = append(c, "")}
    if this.Hint != "" {hint = this.Hint}
    c = append(c, "\x1b[m" + hint)
    return strings.Join(c, "\n")
  } else {
    return fmt.Sprintf(
      "Item: %d %d\n" +
      "Page: %d/%d\n" +
      "Cursor: %d\n" +
      "%v\n",
      item.Offset,
      len(item.Content),
      this.Page + 1,
      len(this.Pages),
      this.Cursor,
      this.PageLen,
    ) + strings.Join(this.Logs, "\n")
  }
}

func (this *EpubViewer) StartProgram() {
  p := tea.NewProgram(this, tea.WithAltScreen())
  if err := p.Start(); err != nil {
    fmt.Println(err)
  }
}
