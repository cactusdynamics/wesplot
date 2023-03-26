wesplot
=======

A real time plotting tool that takes stdin data and pipes it into websocket and
into a JavaScript interactive chart. Usable both on a local computer and on a
remote server.

It's inspired by ttyplot except we leverage the power of the web. Amazing.

Features
--------

### Stdin → browser plotting live streamed or replayed data

### Customizable, interactive plots via command line and GUI

### Simultaneous streams to multiple local and remote devices

### Easy single binary installation

### Drop-in replacement to `ttyplot`

Example use cases
-----------------

### Live system metrics (CPU, memory, IO) plot

<table>
  <tr>
    <th>Metric</th>
    <th>Command</th>
    <th>Comments</th>
  </tr>

  <tr>
    <td>
      CPU usage (Linux via SAR)
    </td>
    <td>
      <code>S_TIME_FORMAT=ISO sar 1 | awk '{ if ($NF ~ /^[0-9]+[.][0-9]*$/) print 100-$NF; fflush(); }' | wesplot -t "CPU Utilization" -c "CPU%" -M 0 -m 100</code>
    </td>
    <td>
    </td>
  </tr>

  <tr>
    <td>
      Memory usage (Linux via SAR)
    </td>
    <td>
      <code>S_TIME_FORMAT=ISO sar -r 2 | awk '{ if ($4 ~ /^[0-9]+$/) print $4/1024; fflush() }' | wesplot -t "Memory usage" -u "MB"</code>
    </td>
    <td>
      Ensure the 4th column of <code>S_TIME_FORMAT=ISO sar -r 2</code> is <code>kbmemused</code>. May be slightly different from output of `free`.
    </td>
  </tr>

  <tr>
    <td>
      Disk usage (read and write)
    </td>
    <td>
      <code>iostat -x 1 | grep --line-buffered nvme0n1 | awk '{ print $3, $9, $15; fflush(); }' | wesplot -t "iostat" -c "Read KB/s" -c "Write KB/s" -c "Discard KB/s" -u "KB/s"</code>
    </td>
    <td>
      This shows the read, write, and discard KB/s for the <code>nvme0n1</code> device. If you are not using a NVME disk, you might need to use a different selector. Replace <code>nvme0n1</code> in the <code>grep</code> clause with your device. To check find device, run <code>iostat</code> and read the output.
    </td>
  </tr>

  <tr>
    <td>
      Network throughput (upload and download)
    </td>
    <td>
      <code>S_TIME_FORMAT=ISO sar -n DEV 1 | awk '$2 == "eth0" { print $5/125, $6/125; fflush(); }' | wesplot -t "Network throughput" -u "Mbit/s" -c "Download" -c "Upload"</code>
    </td>
    <td>
      This monitors the interface <code>eth0</code>. If you want to monitor a different interface, replace the <code>eth0</code> with the interface you wish to monitor.
    </td>
  </tr>

  <tr>
    <td>CPU 0 frequency</td>
    <td>
      <code>{ while true; do cat /sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_cur_freq | awk '{ print $1/1000 }'; sleep 1; done } | wesplot -t "CPU 0 freq" -u "MHz"</code>
    </td>
    <td>
      May need to be root to read <code>cpuinfo_cur_freq</code>. In that case, add <code>sudo</code> before <code>cat</code>.
    </td>
  </tr>

  <tr>
    <td>CPU 0 temperature</td>
    <td>
      <code>{ while true; do awk '{ print $1/1000 }' /sys/class/thermal/thermal_zone0/temp; sleep 1; done } | wesplot -t "CPU 0 temp" -u "C"</code>
      </td>
    <td>Max CPU temperature may be higher as this is only CPU 0.</td>
  </tr>

  <tr>
    <td>Ping latency to 1.1.1.1</td>
    <td>
      <code>ping 1.1.1.1 | sed -u 's/^.*time=//g; s/ ms//g' | wesplot -t 'Ping to 1.1.1.1' -u "ms"</code>
    </td>
    <td>
    </td>
  </tr>

  <tr>
    <td>Firefox memory usage</td>
    <td>
      <code>{ while true; do ps -o rss --sort -rss -p $(pgrep firefox) | head -n2 | awk '{ if ($1 ~ /^[0-9]+$/) print $1/1024; fflush(); }'; sleep 1; done } | wesplot -t "Firefox memory (RSS) usage" -u "MB"</code>
    </td>
    <td>
      If there are multiple <code>firefox</code> processes, this will only show one
    </td>
  </tr>

</table>

### Visualizing metrics on a remote devices (such as a phone)

### Quick data file (CSV) plotting

### Formatting and saving plots for publication

Installation
------------

Features
--------

- [x] Ability to stream data from stdin and plot in the browser
  - [x] There can many time series in a single data stream.
  - [ ] The different series can be comma-separated or space-separated.
  - [x] Can handle infinite streams by caching only the most recent X data points (configurable).
  - [x] Does not lose any messages unless it is expired.
  - [ ] If the data stream ends, the data is cached and can be sent to the browser until the backend is stopped.
    - [ ] This opens the door for plotting a csv file via `cat file.csv | wesplot`.
    - [ ] For streams that are known to end, the backend and frontend cache will effectively be infinite.
  - [ ] Ability to select a column as the timestamp
  - [x] Ability to generate timestamp if doesn't exist in source data
  - [ ] Ability to use non-time values as the x axis.
  - [x] Ability to customize the plot directly from the command line.
  - [ ] Automatically open the local browser upon command.
  - [ ] Proper CORS rules
- [x] Single binary deployment with executable, HTML, CSS, JavaScript all bundled in.
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
  - This means the plot div must be created dynamically (which is mostly easy).
  - Splitting layout is relatively easy with flexbox, but requires a bit of finesse with respect to the height and width and expansion rules.
  - However, resizing the layout is more difficult.
  - Once the layout is resized, exporting and importing the layout is relatively annoying.
  - This also means the backend options becomes meaningless. Things like the title needs to be duplicated multiple times.
    - Likely a config file can be passed to the backend which is then fed to the frontend which contains both the layout and the chart options.


Development setup
-----------------

- Make sure you have Python 3 installed.
- Make sure you install [Go](https://go.dev/).
- Make sure you install [nodejs](https://nodejs.org/en) and [yarn classic](https://classic.yarnpkg.com/en/docs/install) (for now).
- `cd frontend; yarn` to install the frontend dependencies.
- Run `make backend-dev` which will start a development build of wesplot and it will plot a single signal (CPU usage from `sar`).
- In a separate terminal, Run `make frontend-dev` which will start the front end development server.
- Go to http://localhost:5273 to see the frontend.
