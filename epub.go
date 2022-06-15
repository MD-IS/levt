package main

import (
  "io"
  "os"
  "fmt"
  "bytes"
  "regexp"
  "strings"
  "archive/zip"
  "encoding/xml"
)

const (
  MimetypePath  = "mimetype"
  ContainerPath = "META-INF/container.xml"
  TypeXHTML     = "application/xhtml+xml"
  TypeEPUB      = "application/epub+zip"
)

var newLineRe *regexp.Regexp = regexp.MustCompile(
  `[\s\t]*\n[\s\t]*`,
)

type epubContainer struct {
  RootFiles struct {
    Files []struct {
      FullPath  string `xml:"full-path,attr"`
    } `xml:"rootfile"`
  } `xml:"rootfiles"`
}

type epubOPF struct {
  XMLName xml.Name `xml:"package"`

  Spine struct {
    Items []struct {
      Idref string `xml:"idref,attr"`
    } `xml:"itemref"`
  } `xml:"spine"`

  Manifest struct {
    Items []EpubItem `xml:"item"`
  } `xml:"manifest"`

  Metadata struct {
    Title       string  `xml:"title"`
    Rights      string  `xml:"rights"`
    Language    string  `xml:"language"`
    Publisher   string  `xml:"publisher"`
    Creators  []string  `xml:"creator"`
    MetaTags  []struct {
      Name    string  `xml:"name,attr"`
      Content string  `xml:"content,attr"`
    } `xml:"meta"`
  } `xml:"metadata"`

  Base string
}

type EpubItem struct {
  Id    string `xml:"id,attr"`
  Href  string `xml:"href,attr"`
  Type  string `xml:"media-type,attr"`

  Offset  int
  Content map[int]string
  Decoder *xml.Decoder
  Close   func()
}

func (this *EpubItem) Load() {
  reader, _ := openReader(this.Href)
  this.Decoder = xml.NewDecoder(reader)
  this.Content = map[int]string{}
  this.Close = func() {
    this.Decoder = nil
    this.Close = nil
    this.Offset = 0
    reader.Close()
  }
  this.Offset = 0
}

func (this *EpubItem) Line() error {
  o := &this.Offset
  c := this.Content
  d := this.Decoder
  for t, _ := d.Token(); t != nil; {
    switch token := t.(type) {
      case xml.CharData: {
        byt := bytes.Trim(token, "\n\t")
        byt = newLineRe.ReplaceAll(byt, []byte(" "))

        if c[*o] == "" {
          byt = bytes.TrimLeft(byt, " \t")
        }
        if len(byt) != 0 {c[*o] += string(byt)}
      }

      case xml.StartElement: {
        switch token.Name.Local {
          case "p", "div": {
            *o++
            return nil
          }

          case "a": {
            for _, attr := range token.Attr {
              if attr.Name.Local == "href" {
                c[*o] += "##link:" + attr.Value + ";"
              }
            }
            c[*o] += "\x1b[4m"
          }

          case "img", "image": {
            var link string
            alt := "Image"
            for _, attr := range token.Attr {
              atn := attr.Name.Local
              if atn == "alt" {alt = attr.Value}
              if atn == "src" || atn == "href" {
                link = "##link:" + attr.Value + ";"
              }
            }
            if link != "" {
              c[*o] += link +
              "\x1b[1;41m　" + alt + "　\x1b[22;40m"
            }
          }

          case "i", "em": {c[*o] += "\x1b[3m"}
          case "b", "strong": {c[*o] += "\x1b[1m"}
          case "html", "body", "section": {}
          case "head": {d.Skip()}

          case "hr": {
            *o++
            c[*o] += "* * *"
            return nil
          }

          case "h1", "h2", "h3", "h4", "h5", "h6": {
            *o++
            c[*o] += "\x1b[1m"
            return nil
          }

          case "sup": {
            c[*o] += convertSUP(this.Decoder)
          }
          case "sub": {
            c[*o] += convertSUB(this.Decoder)
          }
        }
      }

      case xml.EndElement: {
        switch token.Name.Local {
          case "p", "div", "tr", "li", "html": {
            if c[*o] != "" {
              *o++
              return nil
            }
          }

          case "td": {
            cc := c[*o]
            if len(cc) > 2 {
              if cc[len(cc) - 2] != '|' {
                c[*o] += " | "
              }
            } else {
              c[*o] += " | "
            }
          }

          case "hr", "br": {
            *o++
            return nil
          }

          case "a": {c[*o] += "\x1b[24m"}
          case "i", "em": {c[*o] += "\x1b[23m"}
          case "b", "strong": {c[*o] += "\x1b[22m"}

          case "h1", "h2", "h3", "h4", "h5", "h6": {
            c[*o] += "\x1b[22m"
            *o++
            return nil
          }
        }
      }
    }
    t, _ = d.Token()
  }

  return io.EOF
}

func VerifyMimeType(f *zip.File) {
  if string(ReadContent(f)) == TypeEPUB {return}
  fmt.Println("Invalid Epub file")
  os.Exit(43)
}

func ParseContainer(f *zip.File) (c epubContainer) {
  err := xml.Unmarshal(ReadContent(f), &c)
  if err == nil && c.RootFiles.Files != nil {return}
  fmt.Println("Invalid Epub file")
  os.Exit(44)
  return
}

func ParseOPF(file *zip.File) (opf epubOPF) {
  c := ReadContent(file)
  if err := xml.Unmarshal(c, &opf); err == nil {
    lastSlash := strings.LastIndex(file.Name, "/")
    opf.Base = file.Name[:lastSlash + 1]
    return
  }

  fmt.Println("Invalid Epub file")
  os.Exit(45)
  return
}

func (opf *epubOPF) GetItems() (
  items []EpubItem,
) {
  if len(opf.Spine.Items) == 0 {
    for _, j := range opf.Manifest.Items {
      if j.Type == TypeXHTML {
        j.Href = opf.Base + j.Href
        items = append(items, j)
      }
    }
    return
  }

  for _, i := range opf.Spine.Items {
    for _, j := range opf.Manifest.Items {
      if i.Idref == j.Id {
        j.Href = opf.Base + j.Href
        items = append(items, j)
      }
    }
  }
  return
}
