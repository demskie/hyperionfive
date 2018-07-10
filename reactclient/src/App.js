import React, { Component } from 'react';
//import ReactDOM from 'react-dom';
//import * as Blueprint from '@blueprintjs/core';
import './App.css'; 
import { Button, Navbar, Alignment } from "@blueprintjs/core";

class App extends Component {
  render() {
    return (
      <div className="App">
        <div style={{ position:"fixed",
                      width:"480px",
                      left:"50%",
                      transform:"translateX(-50%)" }}>
          <Navbar className="bp3-dark">
            <Navbar.Group align={Alignment.LEFT}>
              <Button className="bp3-minimal" icon="search-around" text="Systems" />
              <Button className="bp3-minimal" icon="geolocation" text="Fleets" />
              <Navbar.Divider />
              <Button className="bp3-minimal" icon="zoom-in" text="" />
              <Button className="bp3-minimal" icon="zoom-out" text="" />
              <Navbar.Divider />
              <Button className="bp3-minimal" icon="property" text="" />
              <Button className="bp3-minimal" icon="cog" text="" />
              <Navbar.Divider />
              <Button className="bp3-minimal" icon="fullscreen" text="" />
            </Navbar.Group>
          </Navbar>
        </div>
        <canvas id="threeCanvas"
          style={{ backgroundColor:"rgb(0, 0, 0)",
                   overflow:"hidden",
                   top: "0",
                   left: "0",
                   width:"100%vh",
                   height:"100vh" }}>
        </canvas>
      </div>
    );
  }
}

export default App;
