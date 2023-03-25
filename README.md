wesplot
=======

A real time plotting tool that takes stdin data and pipes it into websocket and
into a JavaScript interactive chart. Usable both on a local computer and on a
remote server.

It's inspired by ttyplot except we leverage the power of the web. Amazing.

Features
--------

- [ ] Ability to stream data from stdin and plot in the browser
  - [ ] There can many time series in a single data stream.
  - [ ] The different series can be comma-separated or space-separated.
  - [ ] Can handle infinite streams by caching only the most recent X data points (configurable).
  - [ ] Does not lose any messages unless it is expired.
  - [ ] If the data stream ends, the data is cached and can be sent to the browser until the backend is stopped.
    - [ ] This opens the door for plotting a csv file via `cat file.csv | wesplot`.
    - [ ] For streams that are known to end, the backend and frontend cache will effectively be infinite.
  - [ ] Ability to select a column as the timestamp
  - [ ] Ability to generate timestamp if doesn't exist in source data
  - [ ] Ability to use non-time values as the x axis.
  - [ ] Ability to customize the plot directly from the command line.
  - [ ] Automatically open the local browser upon command.
  - [ ] Proper CORS rules
- [ ] Single binary deployment with executable, HTML, CSS, JavaScript all bundled in.
  - [ ] Mac
  - [ ] Linux
  - [ ] X86
  - [ ] ARM
- [ ] Multiple concurrent browser streaming session
  - [ ] From localhost
  - [ ] From remote hosts
- [ ] UI feature
  - [ ] Pause/resume: when paused, data is continues to be buffered. When resuming, jumps to live.
  - [ ] Stream indicator to indicate if UI is connected to backend and if data stream is not EOF.
  - [ ] Mobile support
  - [ ] Proper error message display
  - [ ] Dark mode
  - [ ] Firefox and Chrome and mobile browsers
  - [ ] Fast on the Raspberry Pi
- [ ] Plot feature
  - [ ] Plot formatting
    - [ ] Title
    - [ ] Axis label
    - [ ] Axis limits
    - [ ] Units
    - [ ] Grid lines
    - [ ] Axis format (time vs not time)
    - [ ] Colors and legends
    - [ ] Tooltip (toggleble)
    - [ ] Line vs bar charts
  - [ ] All plot formatting can be done either via command line from backend or directly in front end.
  - [ ] Zoom, pan, freeze limits, restore to default view
  - [ ] SVG export
  - [ ] Cached size
  - [ ] Consistent lodash versions
  - [ ] Chart.js tree-shaking

### Long term goals

- Backend operator pipelines for easy manipulation (imagine dividing every number by 1024).
- Backend can read binary formats (protobuf, Arrow).
- Backend -> frontend streaming with binary format (arrow/CBOR/whatever)
- Front-end multiple panels for plots.

Development setup
-----------------

- Make sure you have Python 3 installed.
- Make sure you install [Go](https://go.dev/).
- Make sure you install [nodejs](https://nodejs.org/en) and [yarn classic](https://classic.yarnpkg.com/en/docs/install) (for now).
- `cd frontend; yarn` to install the frontend dependencies.
- Run `make backend-dev` which will start a development build of wesplot and it will plot a single signal (CPU usage from `sar`).
- In a separate terminal, Run `make frontend-dev` which will start the front end development server.
- Go to http://localhost:5273 to see the frontend.
