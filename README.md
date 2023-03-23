wesplot
=======

A real time plotting tool that takes stdin data and pipes it into websocket and
into a JavaScript interactive chart. Usable both on a local computer and on a
remote server.

It's inspired by ttyplot except we leverage the power of the web. Amazing.

Development setup
-----------------

- Make sure you have Python 3 installed.
- Make sure you install [Go](https://go.dev/).
- Make sure you install [nodejs](https://nodejs.org/en) and [yarn classic](https://classic.yarnpkg.com/en/docs/install) (for now).
- `cd frontend; yarn` to install the frontend dependencies.
- Run `make backend-dev` which will start a development build of wesplot and it will plot a single signal (CPU usage from `sar`).
- In a separate terminal, Run `make frontend-dev` which will start the front end development server.
- Go to http://localhost:5273 to see the frontend.
