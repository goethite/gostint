import React, { Component } from 'react';
import { render } from 'react-dom';
import {
  Button,
  Col,
  Collapse,
  Container,
  Form,
  FormGroup,
  FormText,
  Input,
  InputGroup,
  Label,
  Row,
  Table
} from 'reactstrap';

import ErrorMsg from '../error_message.js';
import EnvVars from './env_vars.js';
import SecretMaps from './secret_maps.js';

import css from './style.css';

class Action extends Component {
  constructor(props) {
    super(props);

    this.state = {
      advancedForm: false,
      errorMessage: '',
      contentErrorMessage: '',

      dockerImage: '',
      run: '',
      imagePullPolicy: 'IfNotPresent',
      gostintRole: 'gostint-role',
      qName: '',
      content: '',
      entryPoint: '',
      workingDir: '',
      secretFileType: 'yaml',
      contOnWarnings: false,
      secretMaps: [],
      envVars: []
    };
    console.log('in Action this:', this);

    this.advancedSimple = this.advancedSimple.bind(this);
    this.run = this.run.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.handleSecretMaps = this.handleSecretMaps.bind(this);
    this.handleEnvVars = this.handleEnvVars.bind(this);
  }

  handleChange(event) {
    const target = event.target
    console.log('target files:', target.files);
    console.log('target: name:', target.name, 'value:', target.value);
    if (target.name) {
      switch(target.name) {
        case 'contOnWarnings':
          this.setState((state) => {
            return {'contOnWarnings': !state.contOnWarnings};
          });
          break;

        case 'content':
        const content = target.value
          this.setState({contentErrorMessage: ''}, () => {
            this.getBase64(target.files[0])
            .then((data) => {
              console.log('data:', data);
              const parts = data.split(',');
              console.log('parts:', parts);
              if (parts[0].search('gzip') === -1) {
                this.setState({contentErrorMessage: 'not a gzip file'});
                return;
              }
              this.setState((state) => {
                return {
                  'content': content,
                  'contentB64': 'targz,' + parts[1]
                };
              });
            });
          }); // setState contentErrorMessage
          break;

        default:
          this.setState({
            [target.name]: target.value
          });
      }
    }
  }

  render() {
    return (
      <div>
        <Form className={css.form} onSubmit={this.run} name="actionForm">
          <h4 className={css.hdr}>Action</h4>
          <Container>
            <br/>
            <Button
              color="secondary"
              className={css.advButton}
              onClick={this.advancedSimple}>{this.state.advancedForm ? 'Simple' : 'Advanced'}</Button>
            <br/>
            <Row>
              <Col md="3">
                <FormGroup>
                  <Label for="dockerImage">Docker Image:</Label>
                  <Input
                    type="text"
                    name="dockerImage"
                    id="dockerImage"
                    placeholder="Enter docker image:tag"
                    onChange={this.handleChange}
                    value={this.state.dockerImage}
                  />
                </FormGroup>
              </Col>

              <Col md="3">
                <FormGroup>
                  <Label for="run">Container Run Command:</Label>
                  <Input
                    type="text"
                    name="run"
                    id="run"
                    placeholder="Enter A Run Command"
                    onChange={this.handleChange}
                    value={this.state.run}
                  />
                </FormGroup>
              </Col>
            </Row>

            <Collapse isOpen={this.state.advancedForm}>
              <Row>
                <Col md="3">
                  <FormGroup>
                    <Label for="imagePullPolicy">Image Pull Policy:</Label>
                    <Input
                      type="select"
                      name="imagePullPolicy"
                      id="imagePullPolicy"
                      value={this.state.imagePullPolicy}
                      placeholder="Enter Image Pull Policy: 'IfNotPresent' or 'Always'"
                      onChange={this.handleChange}
                    >
                      <option>IfNotPresent</option>
                      <option>Always</option>
                    </Input>
                  </FormGroup>
                </Col>

                <Col md="3">
                  <FormGroup>
                    <Label for="gostintRole">GoStint Vault AppRole Name:</Label>
                    <Input
                      type="text"
                      name="gostintRole"
                      id="gostintRole"
                      value={this.state.gostintRole}
                      placeholder="Enter GoStint Vault AppRole Name"
                      onChange={this.handleChange}
                    />
                  </FormGroup>
                </Col>

                <Col md="3">
                  <FormGroup>
                    <Label for="qName">Queue Name:</Label>
                    <Input
                      type="text"
                      name="qName"
                      id="qName"
                      placeholder="Enter A Queue Name (optional)"
                      onChange={this.handleChange}
                      value={this.state.qName}
                    />
                    <FormText>
                      Queues serialise jobs.  Jobs in different queues run in
                      parallel.
                    </FormText>
                  </FormGroup>
                </Col>
              </Row>

              <FormGroup>
                <Label for="content">Content:</Label>
                <Input
                  type="file"
                  name="content"
                  id="content"
                  onChange={this.handleChange}
                  defaultValue={this.state.content}
                  accept="application/gzip"
                />
                <FormText>
                  Select a Gzipped Tar file of content to be injected into the container for
                  your job.
                </FormText>
                <ErrorMsg>{this.state.contentErrorMessage}</ErrorMsg>
              </FormGroup>

              <Row>
                <Col md="3">
                  <FormGroup>
                    <Label for="entryPoint">Container Entry Point:</Label>
                    <Input
                      type="text"
                      name="entryPoint"
                      id="entryPoint"
                      placeholder="Enter Container Entry Point"
                      onChange={this.handleChange}
                      value={this.state.entryPoint}
                    />
                    <FormText>
                      Leave blank for default entrypoint (optional).
                    </FormText>
                  </FormGroup>
                </Col>

                <Col md="3">
                  <FormGroup>
                    <Label for="workingDir">Container Working Directory:</Label>
                    <Input
                      type="text"
                      name="workingDir"
                      id="workingDir"
                      placeholder="Enter Container Working Directory"
                      onChange={this.handleChange}
                      value={this.state.workingDir}
                    />
                    <FormText>
                      Leave blank for default.
                    </FormText>
                  </FormGroup>
                </Col>

                <Col md="3">
                  <FormGroup>
                    <Label for="secretFileType">Injected Secret File Type:</Label>
                    <Input
                      type="select"
                      name="secretFileType"
                      id="secretFileType"
                      value={this.state.secretFileType}
                      onChange={this.handleChange}
                      >
                        <option>yaml</option>
                        <option>json</option>
                      </Input>
                  </FormGroup>
                </Col>
              </Row>

              <Row>
                <Col>
                  <FormGroup check inline>
                    <Label check>
                      <Input
                        type="checkbox"
                        name="contOnWarnings"
                        id="contOnWarnings"
                        onChange={this.handleChange}
                        checked={this.state.contOnWarnings}
                        value="toggle"
                      />
                      Continue on Warnings
                    </Label>
                  </FormGroup>
                  <FormText>
                  Continue to run job even if vault reported warnings when looking up secret refs.
                  </FormText>
                </Col>
              </Row>

              <SecretMaps secretMaps={this.state.secretMaps} onChange={this.handleSecretMaps} />
              <EnvVars envVars={this.state.envVars} onChange={this.handleEnvVars} />

            </Collapse>

            <br/>
            <Button color="primary" className={css.runButton} type="submit">Run</Button>
            <br/>
            <ErrorMsg>{this.state.errorMessage}</ErrorMsg>
          </Container>
        </Form>
      </div>
    );
  }

  getBase64(file) {
    if (!file) {
      return Promise.resolve('');
    }
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.readAsDataURL(file);
      reader.onload = () => resolve(reader.result);
      reader.onerror = error => reject(error);
    });
  }

  run() {
    console.log('run clicked this:', this);
    event.preventDefault();

    this.setState(() => {
      return {errorMessage: ''};
    }, () => {
      console.log('post action here');

      // see https://github.com/goethite/gostint-client/blob/master/clientapi/clientapi.go

      const job = this.buildJob()
      console.log('job:', job);
      let apiToken;
      let wrapSecretID;
      let encryptedJob;
      let cubbyToken;

      // Get a minimal token for job submission to gostint
      this.vault('v1/auth/token/create', 'POST', {
        policies: ['default'],
        ttl: '1h',
        num_uses: 1,
        display_name: 'gostint_ui'
      })
      .then((res) => {
        console.log('create token res:', res);
        apiToken = res.auth.client_token;

        // Get secret id for gostint approle
        return this.vault(`v1/auth/approle/role/${this.state.gostintRole}/secret-id`, 'POST');
      })
      .then((res) => {
        console.log('get secret_id res:', res);

        // Wrap the secret id
        return this.vault(
          'v1/sys/wrapping/wrap',
          'POST',
          res,
          {'X-Vault-Wrap-TTL': 300}
        );
      })
      .then((res) => {
        console.log('get wrapped secret_id res:', res);
        wrapSecretID = res.wrap_info.token;

        // Encrypt the job payload
        const jobB64 = Buffer.from(JSON.stringify(job)).toString('base64');
        return this.vault(
          `v1/transit/encrypt/${this.state.gostintRole}`,
          'POST',
          {plaintext: jobB64}
        );
      })
      .then((res) => {
        console.log('get encrypted job res:', res);

        encryptedJob = res.data.ciphertext;

        // get limited use token for cubbyhole
        return this.vault('v1/auth/token/create', 'POST', {
          policies: ['default'],
          ttl: '1h',
          use_limit: 2,
          display_name: 'gostint_cubbyhole'
        });
      })
      .then((res) => {
        console.log('cubbyhole token res:', res);
        cubbyToken = res.auth.client_token;

        console.log('encryptedJob:', encryptedJob);

        // Put encrypted job in cubbyhole
        return this.vault(
          'v1/cubbyhole/job',
          'POST',
          {payload: encryptedJob},
          {'X-Vault-Token': cubbyToken}
        );
      })
      .then(() => {
        console.log('cubbyhole post');

        // create job wrapper
        const jWrap = {
          qname: this.state.qName,
          cubby_token: cubbyToken,
          cubby_path: 'cubbyhole/job',
          wrap_secret_id: wrapSecretID
        };

        // submit job
        return this.gostint(
          'v1/api/job',
          'POST',
          jWrap,
          {
            'X-Auth-Token': apiToken,
            'Content-Type': 'application/json'
          }
        );
      })
      .then((res) => {
        console.log('job post res:', res);
      })
      .catch((err) => {
        console.error('err:', err);
        this.setState({errorMessage: err.message});
      })
    });
  }

  buildJob() {
    return {
      qname: this.state.qName,
      container_image: this.state.dockerImage,
      image_pull_policy: this.state.imagePullPolicy,
      content: this.state.contentB64 || '',
      entrypoint: this.state.entryPoint,
      run: this.state.run,
      working_directory: this.state.workingDir,
      env_vars: this.state.envVars,
      secret_refs: this.state.secretMaps,
      secret_file_type: this.state.secretFileType,
      cont_on_warnings: this.state.contOnWarnings
    };
  }

  vault(path, method, data, headers) {
    headers = headers || {}
    if (!headers['X-Vault-Token']) {
      headers['X-Vault-Token'] = this.props.vaultAuth.token;
    }
    console.log('vault: path:', path, 'method:', method, 'data:', data, 'headers:', headers);

    return fetch(this.props.URLs.vault + '/' + path, {
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

  gostint(path, method, data, headers) {
    console.log('gostint: path:', path, 'method:', method, 'data:', data, 'headers:', headers);

    return fetch(this.props.URLs.gostint + '/' + path, {
      headers,
      method: method || 'GET',
      body: data ? JSON.stringify(data) : undefined
    });
  }

  advancedSimple() {
    console.log('in advancedSimple');
    this.setState((state, props) => {
      console.log('advancedSimple state:', state);
      console.log('advancedSimple props:', props);
      return {advancedForm: !state.advancedForm};
    });
  }

  handleSecretMaps(action, row) {
    console.log('in handleSecretMaps, action:', action, 'row:', row);
    switch(action) {
      case 'add':
        this.setState((state, props) => {
          const sr = state.secretMaps
          sr.push(row);
          return {secretMaps: sr};
        });
        break;
      default:
        console.error('handleSecretMaps unrecognised action:', action);
    }
  }

  handleEnvVars(action, row) {
    console.log('in handleEnvVars, action:', action, 'row:', row);
    switch(action) {
      case 'add':
        this.setState((state, props) => {
          const es = state.envVars
          es.push(row);
          return {envVars: es};
        });
        break;
      default:
        console.error('handleEnvVars unrecognised action:', action);
    }
  }
}

export default Action;
