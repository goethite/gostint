import React, { Component } from 'react';
import { render } from 'react-dom';
import {
  Button,
  Pagination,
  PaginationItem,
  PaginationLink,
  Table
} from 'reactstrap';

import { FaTrashAlt } from 'react-icons/fa';

import ErrorMsg from '../error_message.js';
import { gostint } from '../common/gostint_api.js';

import css from './style.css';

class Results extends Component {
  constructor(props) {
    super(props);

    this.state = {
      skip: 0,
      results: {},
      errorMessage: ''
    }

    this.intvl;

    this.selectResult = this.selectResult.bind(this);
    this.resultsPrevious = this.resultsPrevious.bind(this);
    this.resultsNext = this.resultsNext.bind(this);
    this.deleteResult = this.deleteResult.bind(this);
  }

  componentDidMount() {
    this.refreshResults();

    this.intvl = setInterval(() => {
      this.refreshResults();
    }, 10000);
  }

  componentWillUnmount() {
    clearInterval(this.intvl);
  }

  componentWillReceiveProps(props) {
    const { refresh } = this.props;
    if (props.refresh !== refresh) {
      this.refreshResults();
    }
  }

  refreshResults() {
    this.setState(() => {
      return {
        errorMessage: ''
      };
    }, () => {
      gostint(
        this.props.URLs.gostint,
        `v1/api/job?skip=${this.state.skip}`,
        'GET',
        null,
        {
          'X-Auth-Token': this.props.vaultAuth.gostintToken,
          'Content-Type': 'application/json'
        }
      )
      .then((res) => {
        if (res.error) {
          if (res.error.match(/Code: 403/)) {
            return window.location.reload(); // Logout
          }
          return Promise.reject(new Error(res.error));
        }
        this.setState({results: res});
      })
      .catch((err) => {
        console.error('results err:', err);
        this.setState({errorMessage: err.message});
      });
    });
  }

  selectResult(e) {
    const resultId = e.currentTarget.getAttribute('data-item');
    this.props.resultCb(resultId);
  }

  resultsPrevious() {
    this.setState((state) => {
      return {
        skip: state.skip - 10 > 0 ? state.skip - 10 : 0
      };
    }, () => {
      this.refreshResults();
    });
  }

  resultsNext() {
    this.setState((state) => {
      return {
        skip: state.skip + 10 < this.state.results.total ? state.skip + 10 : state.skip
      };
    }, () => {
      this.refreshResults();
    });
  }

  deleteResult(e) {
    event.preventDefault();

    const resultId = e.currentTarget.getAttribute('data-item');
    this.setState(() => {
      return {
        errorMessage: ''
      };
    }, () => {
      gostint(
        this.props.URLs.gostint,
        `v1/api/job/${resultId}`,
        'DELETE',
        null,
        {
          'X-Auth-Token': this.props.vaultAuth.gostintToken,
          'Content-Type': 'application/json'
        }
      )
      .then((res) => {
        this.refreshResults();
      })
      .catch((err) => {
        console.error('results err:', err);
        self.setState({errorMessage: err.message});
      });
    });
  }

  render() {
    return (
      <div className={css.container}>
        <h4 className={css.hdr}>Results</h4>

        <Table striped size="sm">
          <thead>
            <tr>
              <th>ID</th>
              <th>Queue</th>
              <th>Status</th>
              <th>Image</th>
              <th>Submitted</th>
              <th>Started</th>
              <th>Ended</th>
              <th>Return Code</th>
              <th>&nbsp;</th>
            </tr>
          </thead>
          <tbody>
            {this.state.results.data && this.state.results.data.map((r, i) => {
              return (
                <tr
                  className={css.row}
                  key={i.toString()}
                >
                  <td
                    onClick={this.selectResult}
                    data-item={r._id}
                    className={css._id}
                    title="View this job"
                  >{r._id}</td>
                  <td className={css.cell}>{r.qname}</td>
                  <td className={css.cell}>{r.status}</td>
                  <td className={css.cell}>{r.container_image}</td>
                  <td className={css.cell}>{r.submitted}</td>
                  <td className={css.cell}>{r.started}</td>
                  <td className={css.cell}>{r.ended}</td>
                  <td className={css.cell}>{r.return_code}</td>
                  <td>
                    <Button color="danger"
                      className={css.deleteButton}
                      data-item={r._id}
                      title="Delete this job"
                      onClick={this.deleteResult}><FaTrashAlt /></Button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </Table>

        <div className="text-center">
          <Pagination
            className={css.paginationCenter}
            aria-label="Paginate Results">
            <PaginationItem>
              <PaginationLink
                previous
                onClick={this.resultsPrevious}
              />
            </PaginationItem>

            <PaginationItem>
              <PaginationLink
                next
                onClick={this.resultsNext}
              />
            </PaginationItem>
          </Pagination>
        </div>
        <ErrorMsg>{this.state.errorMessage}</ErrorMsg>
      </div>
    );
  }
}

export default Results;
