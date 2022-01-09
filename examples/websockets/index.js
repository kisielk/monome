;(() => {

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


  // expectingMessage is set to true
  // if the user has just submitted a message
  // and so we should scroll the next message into view when received.
  // let expectingMessage = false
  // function dial() {
  //   const conn = new WebSocket(`ws://${location.host}/subscribe`)

  //   conn.addEventListener("close", ev => {
  //     appendLog(`WebSocket Disconnected code: ${ev.code}, reason: ${ev.reason}`, true)
  //     if (ev.code !== 1001) {
  //       appendLog("Reconnecting in 1s", true)
  //       setTimeout(dial, 1000)
  //     }
  //   })
  //   conn.addEventListener("open", ev => {
  //     console.info("websocket connected")
  //   })

  //   // This is where we handle messages received.
  //   conn.addEventListener("message", ev => {
  //     if (typeof ev.data !== "string") {
  //       console.error("unexpected message type", typeof ev.data)
  //       return
  //     }
  //     const p = appendLog(ev.data)
  //     if (expectingMessage) {
  //       p.scrollIntoView()
  //       expectingMessage = false
  //     }
  //   })
  // }
  // dial()

  // const messageLog = document.getElementById("message-log")
  // const publishForm = document.getElementById("publish-form")
  // const messageInput = document.getElementById("message-input")

  // // appendLog appends the passed text to messageLog.
  // function appendLog(text, error) {
  //   const p = document.createElement("p")
  //   // Adding a timestamp to each message makes the log easier to read.
  //   p.innerText = `${new Date().toLocaleTimeString()}: ${text}`
  //   if (error) {
  //     p.style.color = "red"
  //     p.style.fontStyle = "bold"
  //   }
  //   messageLog.append(p)
  //   return p
  // }
  // appendLog("Submit a message to get started!")

  // // onsubmit publishes the message from the user when the form is submitted.
  // publishForm.onsubmit = async ev => {
  //   ev.preventDefault()

  //   const msg = messageInput.value
  //   if (msg === "") {
  //     return
  //   }
  //   messageInput.value = ""

  //   expectingMessage = true
  //   try {
  //     const resp = await fetch("/ledmap", {
  //       method: "POST",
  //       body: msg,
  //     })
  //     if (resp.status !== 202) {
  //       throw new Error(`Unexpected HTTP Status ${resp.status} ${resp.statusText}`)
  //     }
  //   } catch (err) {
  //     appendLog(`Publish failed: ${err.message}`, true)
  //   }
  // }


})()
