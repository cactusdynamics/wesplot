Advanced system metrics
=======================

<table>
  <tr>
    <th>Metric</th>
    <th>Command</th>
    <th>Comments</th>
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
    <td>Firefox memory usage</td>
    <td>
      <code>{ while true; do ps -o rss --sort -rss -p $(pgrep firefox) | head -n2 | awk '{ if ($1 ~ /^[0-9]+$/) print $1/1024; fflush(); }'; sleep 1; done } | wesplot -t "Firefox memory (RSS) usage" -u "MB"</code>
    </td>
    <td>
      If there are multiple <code>firefox</code> processes, this will only show one
    </td>
  </tr>
</table>
