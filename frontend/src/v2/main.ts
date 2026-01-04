/**
 * Wesplot v2 Main Application
 *
 * This is a hello world placeholder for the v2 frontend.
 * It will eventually initialize the Streamer and Chart components.
 */

console.log("Wesplot v2 initialized!");

const app = document.getElementById("app");
if (app) {
  const status = document.createElement("p");
  status.textContent = "✓ v2 frontend is running";
  status.style.color = "green";
  app.appendChild(status);
}
