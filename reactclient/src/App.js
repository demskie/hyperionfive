import React, { Component } from 'react';
//import ReactDOM from 'react-dom';
//import * as Blueprint from '@blueprintjs/core';
import logo from './logo.svg';
import './App.css'; 
import { Button, Navbar, Alignment } from "@blueprintjs/core";

class App extends Component {
  render() {
    return (
      <div className="App">
        <Navbar className="bp3-dark">
          <Navbar.Group align={Alignment.CENTER}>
            <Button className="bp3-minimal" icon="search-around" text="Systems" />
            <Button className="bp3-minimal" icon="geolocation" text="Fleets" />
            <Navbar.Divider />
            <Button className="bp3-minimal" icon="zoom-in" text="" />
            <Button className="bp3-minimal" icon="zoom-out" text="" />
            <Navbar.Divider />
            <Button className="bp3-minimal" icon="settings" text="" />
          </Navbar.Group>
        </Navbar>
        <header className="App-header">
          <img src={logo} className="App-logo" alt="logo" />
          <h1 className="App-title">Welcome to React</h1>
        </header>
        <p className="App-intro">
          To get started, edit <code>src/App.js</code> and save to reload.
        </p>
      </div>
    );
  }
}

export default App;
