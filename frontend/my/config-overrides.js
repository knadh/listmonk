const {injectBabelPlugin} = require("react-app-rewired");
const rewireLess = require("react-app-rewire-less");

module.exports = function override(config, env) {
  config = injectBabelPlugin(
      [
        "import", {
          libraryName: "antd",
          libraryDirectory: "es",
          style: true
        }
      ],  // change importing css to less
      config,
  );
  config = rewireLess.withLoaderOptions({
    modifyVars: {
      "@font-family":
          '"IBM Plex Sans", "Helvetica Neueue", "Segoe UI", "sans-serif"',
      "@font-size-base": "15px",
      "@primary-color": "#7f2aff",
      "@shadow-1-up": "0 -2px 3px @shadow-color",
      "@shadow-1-down": "0 2px 3px @shadow-color",
      "@shadow-1-left": "-2px 0 3px @shadow-color",
      "@shadow-1-right": "2px 0 3px @shadow-color",
      "@shadow-2": "0 2px 6px @shadow-color"
    },
    javascriptEnabled: true,
  })(config, env);
  return config;
};