
export function gostint(URL, path, method, data, headers) {
  return fetch(URL + '/' + path, {
    headers,
    method: method || 'GET',
    body: data ? JSON.stringify(data) : undefined
  });
}
