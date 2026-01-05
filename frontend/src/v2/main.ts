/**
 * Wesplot v2 Test Application
 *
 * This test app uses the Streamer to connect to /ws2 and displays
 * streaming data in tables, one per series.
 */

import type { Metadata } from "../types.js";
import { Streamer } from "./streamer.js";

let baseHost = location.host;
if (import.meta.env.DEV) {
  // This does mean in development, we can only run one of these at a time,
  // which I think is fine.
  baseHost = `${location.hostname}:5274`;
}

const statusDiv = document.getElementById("status") as HTMLDivElement;
const tablesContainer = document.getElementById(
  "tables-container",
) as HTMLDivElement;

function updateStatus(message: string, color: string = "black") {
  if (statusDiv) {
    statusDiv.textContent = message;
    statusDiv.style.color = color;
  }
}

const streamer = new Streamer(`ws://${baseHost}/ws2`, 1000);

const tables: Map<
  number,
  { table: HTMLTableElement; tbody: HTMLTableSectionElement }
> = new Map();

streamer.registerCallbacks({
  onMetadata: (metadata: Metadata) => {
    updateStatus("Connected, received metadata", "green");
    console.log("Metadata:", metadata);

    // Clear existing tables
    tablesContainer.innerHTML = "";
    tables.clear();

    // Set container to flex layout for side-by-side tables
    tablesContainer.style.display = "flex";
    tablesContainer.style.flexDirection = "row";
    tablesContainer.style.gap = "10px";
    tablesContainer.style.flexWrap = "wrap";

    // Create a table for each series
    const numSeries = metadata.WesplotOptions.Columns.length;
    for (let seriesId = 0; seriesId < numSeries; seriesId++) {
      const seriesName =
        metadata.WesplotOptions.Columns[seriesId] || `Series ${seriesId}`;

      const table = document.createElement("table");
      const thead = document.createElement("thead");
      const headerRow = document.createElement("tr");
      const thSeries = document.createElement("th");
      thSeries.textContent = seriesName;
      thSeries.colSpan = 2;
      headerRow.appendChild(thSeries);
      thead.appendChild(headerRow);

      const subHeaderRow = document.createElement("tr");
      const thX = document.createElement("th");
      thX.textContent = "X";
      const thY = document.createElement("th");
      thY.textContent = "Y";
      subHeaderRow.appendChild(thX);
      subHeaderRow.appendChild(thY);
      thead.appendChild(subHeaderRow);

      const tbody = document.createElement("tbody");

      table.appendChild(thead);
      table.appendChild(tbody);

      // Style the table for flex layout
      table.style.flex = "1";
      table.style.minWidth = "200px";

      tablesContainer.appendChild(table);

      tables.set(seriesId, { table, tbody });
    }
  },

  onData: (
    seriesId: number,
    xSegments: Float64Array[],
    ySegments: Float64Array[],
  ) => {
    const tableInfo = tables.get(seriesId);
    if (!tableInfo) return;

    // Concatenate segments
    const xData = concatenateSegments(xSegments);
    const yData = concatenateSegments(ySegments);

    // Clear tbody
    tableInfo.tbody.innerHTML = "";

    // Add rows (last 50, latest at top)
    const maxRows = 50;
    const start = Math.max(0, xData.length - maxRows);
    for (let i = xData.length - 1; i >= start; i--) {
      const row = document.createElement("tr");
      const cellX = document.createElement("td");
      cellX.textContent = xData[i].toString();
      const cellY = document.createElement("td");
      cellY.textContent = yData[i].toString();
      row.appendChild(cellX);
      row.appendChild(cellY);
      tableInfo.tbody.appendChild(row);
    }
  },

  onStreamEnd: (error: boolean, message: string) => {
    const color = error ? "red" : "blue";
    updateStatus(`Stream ended: ${message}`, color);
  },

  onError: (error: Error) => {
    updateStatus(`Error: ${error.message}`, "red");
    console.error(error);
  },
});

// Connect to the streamer
streamer
  .connect()
  .then(() => {
    updateStatus("Connected to WebSocket", "green");
  })
  .catch((error) => {
    updateStatus(`Failed to connect: ${error.message}`, "red");
    console.error(error);
  });

function concatenateSegments(segments: Float64Array[]): Float64Array {
  if (segments.length === 1) {
    return segments[0];
  }
  const totalLength = segments.reduce((sum, seg) => sum + seg.length, 0);
  const result = new Float64Array(totalLength);
  let offset = 0;
  for (const seg of segments) {
    result.set(seg, offset);
    offset += seg.length;
  }
  return result;
}
