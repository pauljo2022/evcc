import axios from "axios";

const { protocol, hostname, port, pathname } = window.location;

const baseAPI = axios.create({
  baseURL: protocol + "//" + hostname + (port ? ":" + port : "") + pathname + "/",
  headers: {
    Accept: "application/json",
  },
});

// global error handling
baseAPI.interceptors.response.use(
  (response) => response,
  (error) => {
    const url = error.config.baseURL + error.config.url;
    const message = `${error.message}: API request failed ${url}`;
    window.app.error({ message });
  }
);

export default baseAPI;
