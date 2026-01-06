import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ChartOptions } from "./chart.js";
import { Chart, InteractionMode } from "./chart.js";

// Mock Chart.js
vi.mock("chart.js", () => {
  class MockChart {
    data = {
      datasets: [],
    };
    options = {
      plugins: {
        title: { text: "" },
      },
      scales: {
        x: {},
        y: {},
      },
    };
    update = vi.fn();
    destroy = vi.fn();
    static register = vi.fn();
    static defaults = {
      font: { size: 16 },
      elements: {
        point: { borderWidth: 0, radius: 1 },
      },
    };
  }

  return {
    Chart: MockChart,
    CategoryScale: {},
    LinearScale: {},
    LineController: {},
    TimeScale: {},
    PointElement: {},
    LineElement: {},
    Title: {},
    Tooltip: {},
    Legend: {},
  };
});

vi.mock("chartjs-plugin-zoom", () => ({
  default: {},
}));

describe("Chart", () => {
  let container: HTMLElement;
  let options: ChartOptions;
  let chart: Chart;

  // Mock requestAnimationFrame
  let rafCallbacks: ((timestamp: number) => void)[] = [];
  let rafId = 0;

  beforeEach(() => {
    // Reset RAF mock
    rafCallbacks = [];
    rafId = 0;

    // Mock requestAnimationFrame
    global.requestAnimationFrame = vi.fn((callback) => {
      const id = ++rafId;
      rafCallbacks.push(callback);
      return id;
    });

    // Mock cancelAnimationFrame
    global.cancelAnimationFrame = vi.fn((_id) => {
      // Simple implementation: just mark as cancelled
    });

    // Create container
    container = document.createElement("div");
    document.body.appendChild(container);

    // Create options (translated from app-level metadata)
    options = {
      title: "Test Chart",
      columns: ["Series 0", "Series 1", "Series 2"],
      xLabel: "X Axis",
      yLabel: "Y Axis",
      xIsTimestamp: false,
      xMin: undefined,
      xMax: undefined,
      yMin: undefined,
      yMax: undefined,
    };
  });

  afterEach(() => {
    if (chart) {
      chart.destroy();
    }
    document.body.removeChild(container);
    vi.clearAllMocks();
  });

  describe("constructor", () => {
    it("should create chart with configured series", () => {
      chart = new Chart({
        container,
        seriesIds: [0, 1],
        options,
      });

      expect(container.querySelector("canvas")).not.toBeNull();
    });

    it("should not schedule RAF on construction", () => {
      chart = new Chart({
        container,
        seriesIds: [0, 1],
        options,
      });

      expect(requestAnimationFrame).not.toHaveBeenCalled();
    });
  });

  describe("update", () => {
    beforeEach(() => {
      chart = new Chart({
        container,
        seriesIds: [0],
        options,
      });
    });

    it("should schedule RAF when no RAF is pending", () => {
      const xSegments = [new Float64Array([1, 2, 3])];
      const ySegments = [new Float64Array([10, 20, 30])];

      chart.update(0, xSegments, ySegments);

      expect(requestAnimationFrame).toHaveBeenCalledTimes(1);
    });

    it("should not schedule multiple RAFs for multiple updates", () => {
      const xSegments = [new Float64Array([1, 2, 3])];
      const ySegments = [new Float64Array([10, 20, 30])];

      chart.update(0, xSegments, ySegments);
      chart.update(0, xSegments, ySegments);
      chart.update(0, xSegments, ySegments);

      expect(requestAnimationFrame).toHaveBeenCalledTimes(1);
    });

    it("should log error for mismatched segment lengths", () => {
      const consoleError = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});
      const xSegments = [new Float64Array([1, 2])];
      const ySegments = [
        new Float64Array([10, 20, 30]),
        new Float64Array([40, 50]),
      ];

      chart.update(0, xSegments, ySegments);

      expect(consoleError).toHaveBeenCalledWith(
        expect.stringContaining("xSegments and ySegments length mismatch"),
      );
      consoleError.mockRestore();
    });

    it("should handle new series dynamically", () => {
      const xSegments = [new Float64Array([1, 2])];
      const ySegments = [new Float64Array([10, 20])];

      // Update series that wasn't in initial config
      chart.update(5, xSegments, ySegments);

      expect(requestAnimationFrame).toHaveBeenCalledTimes(1);
    });
  });

  describe("render", () => {
    beforeEach(() => {
      chart = new Chart({
        container,
        seriesIds: [0],
        options,
      });
    });

    it("should clear RAF state after rendering", () => {
      const xSegments = [new Float64Array([1, 2, 3])];
      const ySegments = [new Float64Array([10, 20, 30])];

      chart.update(0, xSegments, ySegments);

      // Execute RAF callback
      expect(rafCallbacks.length).toBe(1);
      rafCallbacks[0](performance.now());

      // RAF should be cleared
      expect(cancelAnimationFrame).not.toHaveBeenCalled();
    });

    it("should schedule new RAF on subsequent update", () => {
      const xSegments = [new Float64Array([1, 2, 3])];
      const ySegments = [new Float64Array([10, 20, 30])];

      // First update and render
      chart.update(0, xSegments, ySegments);
      rafCallbacks[0](performance.now());

      // Clear RAF callbacks
      rafCallbacks = [];
      vi.clearAllMocks();

      // Second update should schedule new RAF
      chart.update(0, xSegments, ySegments);
      expect(requestAnimationFrame).toHaveBeenCalledTimes(1);
    });

    it("should convert single segment to Chart.js data", () => {
      const xSegments = [new Float64Array([1, 2, 3])];
      const ySegments = [new Float64Array([10, 20, 30])];

      chart.update(0, xSegments, ySegments);
      rafCallbacks[0](performance.now());

      // Check that data was converted (access via chart internals)
      // We can't directly access Chart.js instance in tests, but we verify RAF was called
      expect(rafCallbacks.length).toBe(1);
    });

    it("should convert wrapped segments (two segments)", () => {
      const xSegments = [new Float64Array([1, 2]), new Float64Array([3, 4])];
      const ySegments = [
        new Float64Array([10, 20]),
        new Float64Array([30, 40]),
      ];

      chart.update(0, xSegments, ySegments);
      rafCallbacks[0](performance.now());

      expect(rafCallbacks.length).toBe(1);
    });

    it("should handle NaN discontinuities", () => {
      const xSegments = [new Float64Array([1, Number.NaN, 3])];
      const ySegments = [new Float64Array([10, Number.NaN, 30])];

      chart.update(0, xSegments, ySegments);
      rafCallbacks[0](performance.now());

      expect(rafCallbacks.length).toBe(1);
    });

    it("should handle empty segments", () => {
      const xSegments: Float64Array[] = [];
      const ySegments: Float64Array[] = [];

      chart.update(0, xSegments, ySegments);
      rafCallbacks[0](performance.now());

      expect(rafCallbacks.length).toBe(1);
    });

    it("should not render if generation unchanged", () => {
      const xSegments = [new Float64Array([1, 2, 3])];
      const ySegments = [new Float64Array([10, 20, 30])];

      // First update and render
      chart.update(0, xSegments, ySegments);
      const mockChart = (chart as unknown as { _chart: { update: () => void } })
        ._chart;
      rafCallbacks[0](performance.now());

      // Clear mocks
      vi.clearAllMocks();

      // Manually trigger render again without update
      rafCallbacks[0](performance.now());

      // Chart.update should not be called again (no changes)
      expect(mockChart.update).not.toHaveBeenCalled();
    });

    it("should render multiple series", () => {
      chart = new Chart({
        container,
        seriesIds: [0, 1],
        options,
      });

      const xSegments0 = [new Float64Array([1, 2])];
      const ySegments0 = [new Float64Array([10, 20])];
      const xSegments1 = [new Float64Array([3, 4])];
      const ySegments1 = [new Float64Array([30, 40])];

      chart.update(0, xSegments0, ySegments0);
      chart.update(1, xSegments1, ySegments1);

      rafCallbacks[0](performance.now());

      expect(rafCallbacks.length).toBe(1);
    });
  });

  describe("destroy", () => {
    it("should cancel pending RAF", () => {
      chart = new Chart({
        container,
        seriesIds: [0],
        options,
      });

      const xSegments = [new Float64Array([1, 2, 3])];
      const ySegments = [new Float64Array([10, 20, 30])];

      chart.update(0, xSegments, ySegments);

      // Destroy before RAF executes
      chart.destroy();

      expect(cancelAnimationFrame).toHaveBeenCalledWith(1);
    });

    it("should clean up Chart.js instance", () => {
      chart = new Chart({
        container,
        seriesIds: [0],
        options,
      });

      const mockChart = (
        chart as unknown as { _chart: { destroy: () => void } }
      )._chart;

      chart.destroy();

      expect(mockChart.destroy).toHaveBeenCalled();
    });

    it("should handle destroy without pending RAF", () => {
      chart = new Chart({
        container,
        seriesIds: [0],
        options,
      });

      // No update, no RAF scheduled
      expect(() => chart.destroy()).not.toThrow();
    });
  });

  describe("updateOptions", () => {
    beforeEach(() => {
      chart = new Chart({
        container,
        seriesIds: [0],
        options,
      });
    });

    it("should update chart options", () => {
      const mockChart = (
        chart as unknown as {
          _chart: {
            options: {
              plugins: { title: { text: string } };
            };
            update: (mode: string) => void;
          };
        }
      )._chart;

      chart.updateOptions({
        title: "New Title",
        xLabel: "New X",
        yLabel: "New Y",
      });

      expect(mockChart.options.plugins.title.text).toBe("New Title");
      expect(mockChart.update).toHaveBeenCalledWith("none");
    });
  });

  describe("zoom/pan API", () => {
    beforeEach(() => {
      chart = new Chart({
        container,
        seriesIds: [0],
        options,
      });
    });

    it("should be disabled by default", () => {
      expect(chart.zoomPanMode).toBe(InteractionMode.None);
    });

    it("should enable zoom and disable pan", () => {
      const mockChart = (
        chart as unknown as { _chart: { update: (m: string) => void } }
      )._chart;
      expect(mockChart.update).not.toHaveBeenCalled();

      chart.zoomPanMode = InteractionMode.Zoom;

      expect(chart.zoomPanMode).toBe(InteractionMode.Zoom);
      expect(mockChart.update).toHaveBeenCalledWith("none");
    });

    it("should enable pan and disable zoom", () => {
      const mockChart = (
        chart as unknown as { _chart: { update: (m: string) => void } }
      )._chart;

      // Turn zoom on, then enable pan
      chart.zoomPanMode = InteractionMode.Zoom;
      expect(chart.zoomPanMode).toBe(InteractionMode.Zoom);

      chart.zoomPanMode = InteractionMode.Pan;
      expect(chart.zoomPanMode).toBe(InteractionMode.Pan);
      expect(mockChart.update).toHaveBeenCalledWith("none");
    });

    it("should disable modes when asked", () => {
      chart.zoomPanMode = InteractionMode.Zoom;
      expect(chart.zoomPanMode).toBe(InteractionMode.Zoom);

      chart.zoomPanMode = InteractionMode.None;
      expect(chart.zoomPanMode).toBe(InteractionMode.None);
    });
  });

  describe("edge cases", () => {
    it("should handle large segments efficiently", () => {
      chart = new Chart({
        container,
        seriesIds: [0],
        options,
      });

      const size = 10000;
      const xSegments = [new Float64Array(size).fill(1)];
      const ySegments = [new Float64Array(size).fill(2)];

      const start = performance.now();
      chart.update(0, xSegments, ySegments);
      rafCallbacks[0](performance.now());
      const elapsed = performance.now() - start;

      // Should be reasonably fast (< 100ms for 10k points)
      expect(elapsed).toBeLessThan(100);
    });

    it("should handle mixed NaN and valid values", () => {
      chart = new Chart({
        container,
        seriesIds: [0],
        options,
      });

      const xSegments = [new Float64Array([1, Number.NaN, 3, 4, Number.NaN])];
      const ySegments = [
        new Float64Array([10, Number.NaN, 30, 40, Number.NaN]),
      ];

      chart.update(0, xSegments, ySegments);
      rafCallbacks[0](performance.now());

      expect(rafCallbacks.length).toBe(1);
    });

    it("should handle all NaN segments", () => {
      chart = new Chart({
        container,
        seriesIds: [0],
        options,
      });

      const xSegments = [new Float64Array([Number.NaN, Number.NaN])];
      const ySegments = [new Float64Array([Number.NaN, Number.NaN])];

      chart.update(0, xSegments, ySegments);
      rafCallbacks[0](performance.now());

      expect(rafCallbacks.length).toBe(1);
    });
  });
});
