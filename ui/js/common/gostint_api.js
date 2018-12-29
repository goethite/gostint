
export function gostint(URL, path, method, data, headers) {
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
