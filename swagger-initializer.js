window.onload = () => {
  const params = new URLSearchParams(window.location.search);
  const url = params.get("url") || "/supermq/specs/clients.yaml";
  window.ui = SwaggerUIBundle({
    url: url,
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [SwaggerUIBundle.presets.apis],
    layout: "BaseLayout",
    queryConfigEnabled: true
  });
};
