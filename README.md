wesplot
=======

A live/real-time plotting tool that takes stdin data and pipes it into
websocket and into a JavaScript interactive chart. The backend can run on both
a local computer or on a remote server. The front-end runs in a browser which
means it can run on any device (including mobile devices) that can connect to
the backend.

[demo1.webm](https://user-images.githubusercontent.com/338100/230804080-2396edf2-6744-4a84-ba38-8703b4e10eb4.webm)

Features
--------

### Stdin → browser plotting live streamed or replayed data

Wesplot is designed to work with [Unix
pipelines](https://en.wikipedia.org/wiki/Pipeline_(Unix)). It streams data
written to its standard input (stdin) to one or more browser windows via
[websockets](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API),
where it is then plotted as a scatter plot.

By leveraging the Unix pipeline, there is an endless amount of use cases for
wesplot. Some simple examples are:

1. **Monitor live CPU, memory, IO, network usage**: By using tools like
   [`sar`](https://linux.die.net/man/1/sar), live system usage information can
   be streamed to stdout. If this information is parsed (usually with `awk`),
   it can be piped into wesplot and live system usage can be plotted directly
   against the browser. See [Example use cases](#example-use-cases) for
   some example commands.
2. **Plot CSV data directly from the terminal and generate publication-quality
   plots**: By using [`cat`](https://linux.die.net/man/1/cat), CSV files can be
   piped to wesplot. Wesplot can use values from one column to be the X-axis
   coordinates and the rest of the columns as different series.
3. **Visualize real-time data from remote devices**: Wesplot can be running
   continuously on a remote device such as a server or an IoT device. For
   example, the current network throughput on a server can be piped to a
   persistent wesplot instance which can be connected to and visualized
   remotely. Another example could be an air-quality sensor that pipe its data
   to wesplot which can then be visualized remotely.

### Customizable, interactive plots via command line and GUI

Wesplot is designed to export quality plots suitable for documentations,
presentations, and publications. To support this, the chart title, axis labels,
axis limits, and axis units can be customized both via command-line options as
well as via the settings panel with the browser interface.

### Simultaneous streams to multiple devices

Data piped to wesplot can be visualized simultaneously from multiple browser
tabs and even multiple devices, including mobile devices such as tablets and
phones. One creative use of this is to visualize data coming from a mobile
robot with wesplot on a mobile device as it is being tested in the field.

### Easy single binary installation with cross platform support

Wesplot is designed to be very simple to install. Simply download the
executable and put it in your `$PATH` and you're good to go. It supports all
major platform including Linux, OS X, and Windows for both x86 and ARM.

Installation instruction
------------------------

1. Download the appropriate version of wesplot for your OS and architecture
   from the [latest release](https://github.com/cactusdynamics/wesplot/releases/latest).
2. Rename the executable to `wesplot`.
3. Copy the downloaded executable to a place in your `$PATH`. For example, you
   can copy `wesplot` to `/usr/local/bin`.
4. Make the `wesplot` binary executable via `chmod +x wesplot`.

### Linux-specific instructions

```console
TODO
```

### OSX-specific instructions

1. After following the instructions above, you must execute `wesplot` once.
2. At this point, OS X will open a dialog box that says: "wesplot is from an
   unidentified developer" and prompt you to either cancel or move `wesplot` to
   trash. Click `Cancel`.
3. Open the `System Settings`, go to the `Privacy & Security` in the side bar,
   then go to `Security` (scroll down). You should be able to see a line of
   text that says "wesplot was blocked from opening because it is not from an
   identified developer". Click `Open Anyway` next to that.
4. Launching `wesplot` again from this point on will work.

You can blame Apple for this [feature](https://en.wikipedia.org/wiki/Gatekeeper_(macOS)).

If the above instruction is out-of-date, please consult with Apple's official
documentation on this: https://support.apple.com/en-ca/guide/mac-help/mh40616/mac.

If you don't trust the binary, you can always [build from the source](#building-the-production-binary-from-source).

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
      <code>S_TIME_FORMAT=ISO sar 1 | awk '{ if ($NF ~ /^[0-9]+[.]?[0-9]*$/) print 100-$NF; fflush(); }' | wesplot -t "CPU Utilization" -c "CPU%" -M 0 -m 100</code>
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

### More usage examples

- Advanced system metrics monitoring
- Data file (CSV) plotting
- Use `awk` to process data before piping to wesplot
- Using wesplot with [ROS](ros.org/) (Robot Operating System)

Frequently asked questions (FAQ)
--------------------------------

### How can I format the chart with axis labels, limits, titles, and so on?

### Can I start multiple wesplot sessions?

Yes. Wesplot will automatically find a port starting from 5273 for up to 200
times. If all ports between 5273 and 5473 are taken, you can manually specify a
port via the command line option `--port`. For example: `wesplot --port 1234`
will start wesplot on port 1234.

### Can I view wesplot from multiple browser windows/tabs?

Yes. In fact the browser windows do not even have to reside on the same
computer!

### Can I plot multiple data with multiple columns on the same plot?

### Why am I getting `cannot parse float, ignoring...`?

### How do I set the time value for the data point to be 0 and subsequent data points to be relative from the first?

### How can I plot data whose _x_ values are not time values?

### How can I plot data that already have timestamps as a column?

### How can I plot data from a CSV or TSV file?

### How can I plot data series spanning multiple lines in the same chart?

### How can I save the live data as I'm plotting it?

Development setup
-----------------

- Make sure you have Python 3 installed.
- Make sure you install [Go](https://go.dev/).
- Make sure you install [nodejs](https://nodejs.org/en) and [yarn classic](https://classic.yarnpkg.com/en/docs/install) (for now).
- `cd frontend; yarn` to install the frontend dependencies.
- Run `make backend-dev` which will start a development build of wesplot and it will plot a single signal (CPU usage from `sar`).
- In a separate terminal, Run `make frontend-dev` which will start the front end development server.
- Go to http://localhost:5273 to see the frontend.
  - Note that while you can run multiple wesplots on different ports with the binary, the development setup will only work with a single server as all front-end will listen to the server at the default port (5274).

Building the production binary from source
------------------------------------------

- Make sure you have all development dependencies installed.
- Run `make prod`.
- The resulting binary will be in `build/wesplot`.
