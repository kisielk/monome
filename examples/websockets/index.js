;(() => {

var ws

if (window.WebSocket === undefined) {
  console.error('use a web browser that supports websockets you fossil')
  return;
} else {
  ws = initWS();
}

function initWS() {
  var socket = new WebSocket("ws://localhost:8080/ws")

  socket.addEventListener('open', () => {
    console.log('connection made')
  })
  socket.addEventListener('message', (e) => {
    grid.handleMessage(e.data)
  })
  socket.addEventListener('close', () => {
    console.log('connection closed')
  })

  return socket;
}

//
// create new grid object of size columns/rows inside a wrapper element
//
const newGrid = (columns, rows, wrapperEl) => {
  const wrapper = document.getElementById(wrapperEl)
  const grid = {}

  grid.size = rows*columns
  grid.rows = rows
  grid.columns = columns
  grid.buttons = []
  grid.buffer = {
    offset: {
      x: 0,
      y: 0,
    },
    data: [],
  }
  grid.el = document.createElement('div')

  grid.el.setAttribute('id','grid')
  grid.el.style.gridTemplateColumns = `repeat(${columns}, 1fr)`
  grid.el.style.gridTemplateRows = `repeat(${rows}, 1fr)`

  wrapper.append(grid.el)

  const newButton = (i) => {
    let button = document.createElement('button')
    let [x, y] = grid.getXYFromIndex(i)

    button.classList.add('button_'+i)
    button.classList.add('grid_button')
    button.addEventListener('click', (e) => {grid.clickEvent(x, y, button, e)})

    return button
  }

  grid.getIndexFromXY = (x, y) => {
    return (y*grid.columns) + x
  }

  grid.getXYFromIndex = (i) => {
    let x = i % grid.columns;
    let y = Math.floor(i / grid.columns);
    return [x, y]
  }

  grid.handleMessage = (m) => {
    m = JSON.parse(m)
    switch (m.Cmd) {
      case 'fromGridBuffer':
        grid.buffer.data = m.Data
        grid.render()
    }
  }
  
  grid.clickEvent = (x,y,b,e) => {
    e.preventDefault()
    let i = grid.getIndexFromXY(x,y)
    grid.buffer.data[i] = grid.buffer.data[i] === 0 ? 15 : 0
    let bd = {Cmd: 'levelMap', Data: grid.buffer.data}
    grid.render()
    ws.send(JSON.stringify(bd))
  }

  grid.paint = () => {
    for (let i = 0; i < grid.size; i++) {
      grid.buffer.data[i] = 0
      grid.buttons[i] = newButton(i)
      grid.el.append(grid.buttons[i])
    }
    grid.el.classList.add('visible')
    grid.width = grid.el.clientWidth
    grid.height = grid.el.clientHeight
  }

  grid.hide = () => {
    grid.el.classList.del('visible')
  }

  grid.render = () => {
    grid.buffer.data.forEach((b,i) => {
      grid.buttons[i].value = b
      if (b == 0) { 
        grid.buttons[i].style.background = 'rgb(145, 145, 145)'
      } else {
        grid.buttons[i].style.background = `rgba(255, 255, 255, ${b/15})`
      }
    })
  }

  return grid
}

//
// initialize
//
const grid = newGrid(16, 8, 'wrapper')
grid.paint()
grid.render()

})()
