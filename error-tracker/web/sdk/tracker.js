(function (global) {
  "use strict";

  var config = {
    appId: "",
    reportUrl: "",
    userId: "",
    maxBatchSize: 10,
    flushInterval: 5000,
  };

  var queue = [];
  var timer = null;

  function init(options) {
    if (!options.appId || !options.reportUrl) {
      console.error("[ErrorTracker] appId and reportUrl are required");
      return;
    }
    config.appId = options.appId;
    config.reportUrl = options.reportUrl;
    config.userId = options.userId || "";

    setupListeners();
    startFlushTimer();
  }

  function setupListeners() {
    // JS runtime errors
    global.onerror = function (message, source, lineno, colno, error) {
      pushEvent({
        type: "js_error",
        message: message,
        stack: error ? error.stack || "" : "",
        url: source || location.href,
        meta: JSON.stringify({ lineno: lineno, colno: colno }),
      });
    };

    // Unhandled promise rejections
    global.addEventListener("unhandledrejection", function (event) {
      var message = "Unhandled rejection";
      var stack = "";
      if (event.reason) {
        message = event.reason.message || String(event.reason);
        stack = event.reason.stack || "";
      }
      pushEvent({
        type: "unhandled_rejection",
        message: message,
        stack: stack,
        url: location.href,
      });
    });

    // Resource load errors (images, scripts, etc.)
    global.addEventListener(
      "error",
      function (event) {
        var target = event.target || event.srcElement;
        if (
          target &&
          target !== global &&
          (target.tagName === "IMG" ||
            target.tagName === "SCRIPT" ||
            target.tagName === "LINK")
        ) {
          pushEvent({
            type: "resource_error",
            message: target.tagName + " load failed",
            stack: "",
            url: target.src || target.href || "",
            meta: JSON.stringify({ tagName: target.tagName }),
          });
        }
      },
      true
    );

    // Flush remaining on page unload
    global.addEventListener("beforeunload", function () {
      flush(true);
    });
  }

  function pushEvent(data) {
    var event = {
      app_id: config.appId,
      type: data.type,
      message: data.message,
      stack: data.stack || "",
      url: data.url || location.href,
      user_id: config.userId,
      meta: data.meta || "",
      timestamp: Date.now(),
    };
    queue.push(event);

    if (queue.length >= config.maxBatchSize) {
      flush(false);
    }
  }

  function flush(useBeacon) {
    if (queue.length === 0) return;

    var events = queue.splice(0);
    var body = JSON.stringify(events);

    if (useBeacon && navigator.sendBeacon) {
      navigator.sendBeacon(config.reportUrl, body);
      return;
    }

    var xhr = new XMLHttpRequest();
    xhr.open("POST", config.reportUrl, true);
    xhr.setRequestHeader("Content-Type", "application/json");
    xhr.onloadend = function () {
      if (xhr.status !== 200) {
        // retry once
        var retry = new XMLHttpRequest();
        retry.open("POST", config.reportUrl, true);
        retry.setRequestHeader("Content-Type", "application/json");
        retry.send(body);
      }
    };
    xhr.send(body);
  }

  function startFlushTimer() {
    if (timer) clearInterval(timer);
    timer = setInterval(function () {
      flush(false);
    }, config.flushInterval);
  }

  function captureError(error) {
    pushEvent({
      type: "js_error",
      message: error.message || String(error),
      stack: error.stack || "",
      url: location.href,
    });
  }

  global.ErrorTracker = {
    init: init,
    captureError: captureError,
  };
})(window);
