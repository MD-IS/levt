package main

import(
  "os"
  "fmt"
  "path"
  "strings"
  "strconv"
  "archive/zip"
  _ "embed"
)

//go:embed version.txt
var version string

var epubContent = make(map[string]*zip.File)

func printHelp() {
  fmt.Printf(`
LEVT Version %s
Usage: %s [-f] <path/to/file.epub> [-i pagenumber]  

Flags:
  -h: Print this message
  -lf <to/file.epub>: List content of <file.epub>
  -mf <to/file.epub>: Print Metadata of <file.epub>

`, version, os.Args[0])
}

func parseOpt(args []string) map[rune]string {
  opt := make(map[rune]string)
  for i, v := range args {
    if v == "--" {break}
    if v[0] != '-' {continue}
    for _, c := range v[1:] {
      if (i + 1) < len(args) {
        optValue := args[i + 1]
        if optValue == "--" && (i + 2) < len(args) {
          optValue = args[i + 2]
        }
        opt[c] = optValue
        continue
      }
      opt[c] = "-"
    }
  }

  return opt
}

func main() {
  args := os.Args[1:]
  opt := parseOpt(args)
  if _, ok := opt['h']; ok {
    printHelp()
    os.Exit(0)
  }

  cursor := 0
  index := 0

  epubPath := opt['f']
  htmlPath := opt['F']
  optI, ok2 := opt['i']

  if ok2 {
    optIint, err := strconv.Atoi(optI)
    if err != nil || optIint < 1 {
      fmt.Println("Warning: -i flag need positive non-zero number as argument")
    } else {
      index = optIint - 1
    }
  }

  if (epubPath == "") && (len(args) > 0) &&
      (args[0][0] != '-') {epubPath = args[0]}

  if epubPath == "" && htmlPath == "" {
    config, err := getConfig()
    if err != nil {
      fmt.Println(err)
      os.Exit(17)
    }
    c := config.Bookmarks
    if len(c) == 0 && config.LastRead.FilePath == "" {
      printHelp()
      os.Exit(0)
    }

    config.onExit = func(i int) {
      if i < 0 {return}
      var sc StartConf

      if i >= len(c) {
        sc = config.LastRead
      } else {
        sc = c[i]
      }

      cursor = sc.Cursor
      if sc.IsXHTML {
        htmlPath = sc.FilePath
      } else {
        epubPath = sc.FilePath
        index = sc.Index
      }
    }

    err = config.StartConfig()
    if err != nil {
      fmt.Println(err)
      os.Exit(18)
    }

    if epubPath == "" && htmlPath == "" {os.Exit(0)}
  }

  if epubPath == "-" {
    epubPath = path.Join(os.TempDir(), "STDIN")
    SaveTMP(os.Stdin, epubPath)
  }

  if htmlPath == "-" {
    htmlPath = path.Join(os.TempDir(), "STDIN")
    SaveTMP(os.Stdin, htmlPath)
  }

  if htmlPath != "" {
    (&EpubViewer{
      Cursor: cursor,
      FilePath: htmlPath,
      EpubItems: []EpubItem{{Href: htmlPath}},
    }).StartProgram()
    os.Exit(0)
  }

  reader, err := zip.OpenReader(epubPath)
  if err != nil {
    fmt.Println(err)
    os.Exit(2)
  }
  defer reader.Close()

  for _, file := range reader.File {
    epubContent[file.Name] = file
  }

  VerifyMimeType(epubContent[MimetypePath])
  cont := ParseContainer(epubContent[ContainerPath])
  opfPath := cont.RootFiles.Files[0].FullPath
  opf := ParseOPF(epubContent[opfPath])

  if arg, ok := opt['m']; ok {
    // (-m) Print Metadata then Exit
    fmt.Printf(
      "Title: %s\n" +
      "Author: %s\n",
      opf.Metadata.Title,
      strings.Join(opf.Metadata.Creators, ", "),
    )

    publisher := opf.Metadata.Publisher
    if publisher != "" {
      fmt.Printf("Publisher: %s\n", publisher)
    }

    rights := opf.Metadata.Rights
    if rights != "" {
      fmt.Printf("Rights: %s\n", rights)
    }

    loc := opf.Metadata.Language
    if loc != "" {
      fmt.Printf("Locale: %s\n", loc)
    }

    for _, m := range opf.Metadata.MetaTags {
      if m.Name == "cover" {
        if arg == "Cover" {
          var href string
          lastSlash := strings.LastIndex(opfPath, "/")
          opfBase := opfPath[:lastSlash + 1]
          for _, item := range opf.Manifest.Items {
            if item.Id == m.Content {href = item.Href}
          }
          if href != "" {
            open(opfBase + href)
          } else {
            fmt.Println("\nnot found", m.Content)
          }
        } else {
          fmt.Println("\nuse '-m Cover' for cover")
        }
      }
    }
    os.Exit(0)
  }

  items := opf.GetItems()
  if len(items) < 1 {
    fmt.Printf(
      "Empty Epub, file %s has no items\n", epubPath,
    )
    os.Exit(2)
  }

  if index >= len(items) {
    fmt.Printf(
      "Out of Range, file %s has only %d items\n",
      epubPath, len(items),
    )
    os.Exit(3)
  }

  if _, ok := opt['l']; ok {
    for _, item := range items {
      fmt.Println(item.Href)
    }
    os.Exit(0)
  }

  optP, ok := opt['p']
  if ok && optP != "" {
    index = 0
    for i, item := range items {
      if item.Href == optP {
        index = i + 1
      }
    }
    if index < 1 {
      open(optP)
      os.Exit(0)
    }
  }

  (&EpubViewer{
    Index: index,
    Cursor: cursor,
    FilePath: epubPath,
    EpubItems: items,
    EPUBTitle: opf.Metadata.Title,
  }).StartProgram()
}
