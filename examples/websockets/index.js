
//
// create new grid object of size columns/rows inside a wrapper element
//
const newGrid = (columns, rows, wrapperEl) => {
  const wrapper = document.getElementById(wrapperEl)
  const grid = {}

  grid.clickEvent = () => {}
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
    let x = i % grid.columns;
    let y = Math.floor(i / grid.columns);
    let button = document.createElement('button')

    button.classList.add('button_'+i)
    button.classList.add('grid_button')
    button.addEventListener('click', (e) => {grid.clickEvent(x, y, button, e)})

    return button
  }

  grid.paint = () => {
    for (let i = 0; i < grid.size; i++) {
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

grid.clickEvent = (x,y,t) => {
  console.log(x, y, t)
}

randomizeBuffer = (grid) => {
  for (let i = 0; i < grid.size; i++) {
    grid.buffer.data[i] = Math.floor(Math.random() * 15)
  }
}

randomizeBuffer(grid)
grid.render()

