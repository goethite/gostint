const path = require('path');

module.exports = {
  entry: './js/app.js',
  output: {
    filename: 'bundle.js',
    path: path.resolve(__dirname, 'dist')
  },
  devtool: 'cheap-module-source-map',
  module: {
    rules: [
      {
        test: /\.s?css$/,
        loaders: ['to-string-loader', 'css-loader', 'sass-loader']
      },
      { test: /\.html$/, loader: 'html-loader' },
      {
        test: /\.js$/,
        exclude: /node_modules/,
        use: {
          loader: "babel-loader",
          options: {
            presets: ['@babel/preset-env', '@babel/preset-react']
          }
        }
      }
    ]
  }
};
