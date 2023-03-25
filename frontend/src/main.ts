import "./app.css";
import "@fortawesome/fontawesome-free/css/fontawesome.css";
import "@fortawesome/fontawesome-free/css/solid.css";
import { merge } from "lodash";
import { Chart, ChartConfiguration } from "chart.js/auto";

type DataRow = {
  Timestamp: number;
  Data: number[];
};

interface DataItem {
  value: [number, number];
}

interface Metadata {
  WindowSize: number;
  Columns: string[];
  YUnit: string;
  // EChartsOption: echarts.EChartsOption;
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
            text: "X label",
          },
        },
        y: {
          beginAtZero: true,
          title: {
            display: true,
            text: "Y label",
          },
          ticks: {
            // Include a unit
            callback: (value, _index, _ticks) => `${value} bananas`,
          },
        },
      },
      plugins: {
        title: {
          display: true,
          text: "Title",
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
