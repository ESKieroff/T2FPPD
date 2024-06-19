package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

type Server struct {
    mutex   sync.Mutex
    matrix  [][]Elemento
    clients map[string]*ClientInfo
}

func NewServer(matrix [][]Elemento) *Server {
    return &Server{
        clients: make(map[string]*ClientInfo),
        matrix:  matrix,
    }
}

type RegisterArgs struct {
    Name string
}

type UpdateArgs struct {
    Name string
    X    int
    Y    int
}

type ClientInfo struct {
    Name       string
    X          int
    Y          int
    UpdateChan chan [][]Elemento
}

type RegisterResponse struct {
    X      int
    Y      int
    Matrix [][]Elemento
}

type Elemento struct {
    Simbolo  rune
    Cor      termbox.Attribute
    CorFundo termbox.Attribute
    Tangivel bool
}

// Personagem controlado pelo jogador
var personagem = Elemento{
    Simbolo:  '☺',
    Cor:      termbox.ColorBlack,
    CorFundo: termbox.ColorDefault,
    Tangivel: true,
}

// Parede
var parede = Elemento{
    Simbolo:  '▤',
    Cor:      termbox.ColorBlack | termbox.AttrBold | termbox.AttrDim,
    CorFundo: termbox.ColorDarkGray,
    Tangivel: true,
}

// Barrreira
var barreira = Elemento{
    Simbolo:  '#',
    Cor:      termbox.ColorRed,
    CorFundo: termbox.ColorDefault,
    Tangivel: true,
}

// Vegetação
var vegetacao = Elemento{
    Simbolo:  '♣',
    Cor:      termbox.ColorGreen,
    CorFundo: termbox.ColorDefault,
    Tangivel: false,
}

// Elemento vazio
var vazio = Elemento{
    Simbolo:  ' ',
    Cor:      termbox.ColorDefault,
    CorFundo: termbox.ColorDefault,
    Tangivel: false,
}

// Elemento para representar áreas não reveladas (efeito de neblina)
var neblina = Elemento{
    Simbolo:  '.',
    Cor:      termbox.ColorDefault,
    CorFundo: termbox.ColorYellow,
    Tangivel: false,
}
var ultimoElementoSobPersonagem = vazio

func (server *Server) RegisterClient(args *RegisterArgs, response *RegisterResponse) error {
    server.mutex.Lock()
    defer server.mutex.Unlock()

    if _, exists := server.clients[args.Name]; exists {
        return fmt.Errorf("cliente com o nome %s já está registrado", args.Name)
    }

    rand.Seed(time.Now().UnixNano())
    boolFlag := false
    posX := 0
    posY := 0

    for !boolFlag {
        posX = rand.Intn(len(server.matrix[0]))
        posY = rand.Intn(len(server.matrix))

        if server.matrix[posY][posX] == vazio {
            boolFlag = true
        }
    }

    updateChan := make(chan [][]Elemento, 1) // Canal bufferizado
    server.clients[args.Name] = &ClientInfo{
        Name:       args.Name,
        X:          posX,
        Y:          posY,
        UpdateChan: updateChan,
    }
    server.matrix[posY][posX] = personagem
    response.X = posX
    response.Y = posY
    response.Matrix = server.matrix
    fmt.Printf("Cliente %s cadastrado!\n", args.Name)
    return nil
}

func (server *Server) UpdateClientPosition(args *UpdateArgs, ack *bool) error {
    server.mutex.Lock()
    defer server.mutex.Unlock()

    client, exists := server.clients[args.Name]
    if !exists {
        *ack = false
        return fmt.Errorf("cliente com o nome %s não está registrado", args.Name)
    }

    if args.X > 0 && args.Y < len(server.matrix) && args.X >= 0 && args.X < len(server.matrix[args.Y]) && server.matrix[args.Y][args.X].Tangivel == false {
        server.matrix[client.Y][client.X] = ultimoElementoSobPersonagem
        server.matrix[args.Y][args.X] = personagem
        client.X = args.X
        client.Y = args.Y

        for _, client := range server.clients {
            // clearChannel(client.UpdateChan)
            select {
            case client.UpdateChan <- server.matrix:
            default:
                fmt.Printf("Canal do cliente %s está cheio, ignorando atualização\n", client.Name)
            }
        }
    }

    *ack = true
    return nil
}

func (server *Server) GetUpdates(name string, response *RegisterResponse) error {
    server.mutex.Lock()
    client, exists := server.clients[name]
    server.mutex.Unlock()

    if !exists {
        return fmt.Errorf("cliente com o nome %s não está registrado", name)
    }

    // clientArgs.X = client.X
    // clientArgs.Y = client.Y
    response.Matrix = <-client.UpdateChan
    response.X = client.X
    response.Y = client.Y

    return nil
}

func main() {
    matrix, err := carregarMapa("mapa.txt")
    if err != nil {
        log.Fatalf("Erro ao carregar o mapa: %v", err)
    }

    server := NewServer(matrix)
    rpc.Register(server)

    listener, err := net.Listen("tcp", ":1234")
    if err != nil {
        log.Fatalf("Erro ao iniciar o listener: %v", err)
    }
    defer listener.Close()

    log.Println("Servidor iniciado na porta :1234...")

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Erro ao aceitar conexão: %v", err)
            continue
        }

        //go rpc.ServeConn(conn)
        go func() {
            rpc.ServeConn(conn)
            conn.Close() // Fecha a conexão após o término do RPC
        }()
    }
}

func carregarMapa(nomeArquivo string) ([][]Elemento, error) {
    arquivo, err := os.Open(nomeArquivo)
    if err != nil {
        return nil, err
    }
    defer arquivo.Close()

    scanner := bufio.NewScanner(arquivo)
    var matrix [][]Elemento

    for scanner.Scan() {
        linhaTexto := scanner.Text()
        var linha []Elemento
        for _, char := range linhaTexto {
            elementoAtual := vazio
            switch char {
            case parede.Simbolo:
                elementoAtual = parede
            case barreira.Simbolo:
                elementoAtual = barreira
            case vegetacao.Simbolo:
                elementoAtual = vegetacao
            case personagem.Simbolo:
                //elementoAtual = vazio
                elementoAtual = personagem // pra evitar que zero seja o personagem
            default:    // vazio
                elementoAtual = vazio
            }
            linha = append(linha, elementoAtual)
        }
        matrix = append(matrix, linha)
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return matrix, nil
}


func clearChannel(ch chan [][]Elemento) {
    for {
        select {
        case <-ch:
        default:
            return
        }
    }
}