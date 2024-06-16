package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"

	"github.com/nsf/termbox-go"
)

type Elemento struct {
    Simbolo  rune
    Cor      termbox.Attribute
    CorFundo termbox.Attribute
    Tangivel bool
}

type ClientArgs struct {
	Name string
	X    int
	Y    int
}

type RegisterArgs struct {
	Name string
}

type RegisterResponse struct {
    X      int
    Y      int 
	Matrix [][]Elemento
}

type GetResponse struct{
	Name string
	X    int
	Y    int
	Matrix [][]Elemento
}

var personagem = Elemento{
    Simbolo:  '☺',
    Cor:      termbox.ColorBlack,
    CorFundo: termbox.ColorDefault,
    Tangivel: true,
}

var parede = Elemento{
    Simbolo:  '▤',
    Cor:      termbox.ColorBlack | termbox.AttrBold | termbox.AttrDim,
    CorFundo: termbox.ColorDarkGray,
    Tangivel: true,
}

var barreira = Elemento{
    Simbolo:  '#',
    Cor:      termbox.ColorRed,
    CorFundo: termbox.ColorDefault,
    Tangivel: true,
}

var vegetacao = Elemento{
    Simbolo:  '♣',
    Cor:      termbox.ColorGreen,
    CorFundo: termbox.ColorDefault,
    Tangivel: false,
}

var vazio = Elemento{
    Simbolo:  ' ',
    Cor:      termbox.ColorDefault,
    CorFundo: termbox.ColorDefault,
    Tangivel: false,
}

var neblina = Elemento{
    Simbolo:  '.',
    Cor:      termbox.ColorDefault,
    CorFundo: termbox.ColorYellow,
    Tangivel: false,
}

var mapa [][]Elemento
var clientArgs ClientArgs

var statusMsg string

var efeitoNeblina = false
var revelado [][]bool
var raioVisao int = 3

func registerClient(conn *rpc.Client, name string) error {
	var response RegisterResponse
	args := &RegisterArgs{Name: name}
	err := conn.Call("Server.RegisterClient", args, &response)
	if err != nil {
		return fmt.Errorf("erro ao registrar cliente: %v", err)
	}

	mapa = response.Matrix
	clientArgs.X = response.X
	clientArgs.Y = response.Y

	fmt.Printf("Cliente %s registrado com sucesso!\n", name)
	return nil
}

func getUpdates(conn *rpc.Client, args *ClientArgs) error {
	var response RegisterResponse
	for {
		err := conn.Call("Server.GetUpdates", args.Name, &response)
		if err != nil {
			return fmt.Errorf("erro ao obter atualizações: %v", err)
		}

		if efeitoNeblina {
			revelarArea()
		}

		clientArgs.X = response.X
		clientArgs.Y = response.Y
		mapa         = response.Matrix
		desenhaTudo()
	}
}

func updateClientPosition(conn *rpc.Client, args *ClientArgs, X, Y int) error {
	var ack bool
	clientForLoadUpdate := ClientArgs{
		Name: args.Name,
		X:    args.X + X,
		Y:    args.Y + Y,
	}
	err := conn.Call("Server.UpdateClientPosition", &clientForLoadUpdate, &ack)
	if err != nil {
		return fmt.Errorf("erro ao atualizar posição do cliente: %v", err)
	}
	if !ack {
		return fmt.Errorf("falha ao atualizar posição do cliente %s", args.Name)
	}
	return nil
}

func main() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	servidor := os.Args[1]
	nome := os.Args[2]    
	porta := 1234
	conn, err := rpc.Dial("tcp", fmt.Sprintf("%s:%d", servidor, porta))
	if err != nil {
		log.Fatalf("Erro ao conectar ao servidor: %v", err)
	}
	defer conn.Close()

	registerClient(conn, nome)

	if efeitoNeblina {
		revelarArea()
	}
	clientArgs.Name = nome
	desenhaTudo()
	go getUpdates(conn, &clientArgs)

	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyEsc {
				return // Sair do programa
			}
			if ev.Ch == 'e' {
				//interagir()
			} else {
				mover(ev.Ch, conn)
			}
		}
	}
}

func mover(comando rune, conn *rpc.Client) {
	dx, dy := 0, 0
	switch comando {
	case 'w':
		dy = -1
	case 'a':
		dx = -1
	case 's':
		dy = 1
	case 'd':
		dx = 1
	}

	updateClientPosition(conn, &clientArgs, dx, dy)
}

func revelarArea() {
	altura := len(mapa)
	largura := len(mapa[0])
	revelado = make([][]bool, altura)
	for i := range revelado {
		revelado[i] = make([]bool, largura)
	}

	for dy := -raioVisao; dy <= raioVisao; dy++ {
		for dx := -raioVisao; dx <= raioVisao; dx++ {
			ny := clientArgs.Y + dy
			nx := clientArgs.X + dx
			if ny >= 0 && ny < altura && nx >= 0 && nx < largura {
				revelado[ny][nx] = true
			}
		}
	}
}

func desenhaTudo() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	for y, linha := range mapa {
		for x, elemento := range linha {
			if efeitoNeblina && !revelado[y][x] {
				termbox.SetCell(x, y, neblina.Simbolo, neblina.Cor, neblina.CorFundo)
			} else {
				termbox.SetCell(x, y, elemento.Simbolo, elemento.Cor, elemento.CorFundo)
			}
		}
	}
	termbox.Flush()
}