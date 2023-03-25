import "./app.css";
import "@fortawesome/fontawesome-free/css/fontawesome.css";
import "@fortawesome/fontawesome-free/css/solid.css";
import { merge } from "lodash";
import { Chart, ChartConfiguration } from "chart.js/auto";
import zoomPlugin from "chartjs-plugin-zoom";
import "chartjs-adapter-date-fns";

Chart.register(zoomPlugin);

type DataRow = {
  Timestamp: number;
  Data: number[];
};

interface ChartOptions {
  Title: string;
  XLabel: string;
  YLabel: string;
  YMin: number;
  YMax: number;
}

interface Metadata {
  WindowSize: number;
  Columns: string[];
  YUnit: string;
  ChartOptions: ChartOptions;
}

async function main() {
  const response = await fetch(`http://${location.hostname}:8080/metadata`);
  const metadata: Metadata = await response.json();
  const ctx = document.getElementById("myChart")! as HTMLCanvasElement;

  const config: ChartConfiguration<"scatter"> = {
    type: "scatter",
    data: {
      datasets: [],
    },
    options: {
      showLine: true,
      maintainAspectRatio: false,
      scales: {
        x: {
          title: {
            display: true,
            text: metadata.ChartOptions.XLabel,
          },
          type: "time",
        },
        y: {
          beginAtZero: true,
          title: {
            display: true,
            text: metadata.ChartOptions.YLabel,
          },
          ticks: {
            // Include a unit
            callback: (value, _index, _ticks) => {
              if (!metadata.YUnit) {
                return value;
              }
              if (typeof value === "number") {
                return `${value.toFixed(3)} ${metadata.YUnit}`; // TODO: fix this
              }
              return `${value} ${metadata.YUnit}`;
            },
            precision: 3,
          },
        },
      },
      plugins: {
        title: {
          display: true,
          text: metadata.ChartOptions.Title,
          font: {
            size: 20,
          },
        },
        tooltip: {
          enabled: true,
        },
        legend: {
          position: "bottom",
        },
        zoom: {
          zoom: {
            wheel: {
              enabled: true,
            },
            pinch: {
              enabled: true,
            },
            mode: "xy",
          },
          pan: {
            enabled: true,
            mode: "xy",
          },
        },
      },
    },
  };
  Chart.defaults.font.size = 16;
  for (const column of metadata.Columns) {
    config.data.datasets.push({
      label: column,
      data: [],
      borderWidth: 1,
    });
  }
  const chart = new Chart(ctx, config);

  const hostname = `ws://${location.hostname}:8080/ws`;
  console.log(`connecting to ${hostname}`);
  const socket = new WebSocket(hostname);

  socket.addEventListener("open", () => {
    console.log("Successfully Connected");
  });

  socket.addEventListener("close", (event) => {
    console.log("Socket Closed Connection: ", event);
  });

  socket.addEventListener("error", (error) => {
    console.log("Socket Error: ", error);
  });

  socket.addEventListener("message", (event) => {
    const rows: DataRow[] = JSON.parse(event.data);
    for (const [i, _] of metadata.Columns.entries()) {
      const data = chart.data.datasets[i].data;
      for (const row of rows) {
        data.push({ x: row.Timestamp, y: row.Data[i] });
        chart.data.datasets[i].data = data;
        if (data.length > metadata.WindowSize) {
          data.shift();
        }
      }
    }
    chart.update("none");
  });
}

window.addEventListener("load", main);
