
export function vault(URL, token, path, method, data, headers) {
  headers = headers || {}
  if (!headers['X-Vault-Token']) {
    headers['X-Vault-Token'] = token;
  }

  return fetch(URL + '/' + path, {
    headers,
    method: method || 'GET',
    body: data ? JSON.stringify(data) : undefined
  })
  .then((res) => {
    switch(res.status) {
      case 200:
        return res.json();
      case 204:
        return;
      default:
        return Promise.reject(
          new Error(`Request failed with status: ${res.status} ${res.statusText} [${res.url}]`)
        );
    }
  });
}
