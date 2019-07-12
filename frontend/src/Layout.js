import React from "react";
import { Switch, Route } from "react-router-dom";
import { Link } from "react-router-dom";
import { Layout, Menu, Icon } from "antd";

import logo from "./static/listmonk.svg";

// Views.
import Dashboard from "./Dashboard";
import Lists from "./Lists";
import Subscribers from "./Subscribers";
import Subscriber from "./Subscriber";
import Templates from "./Templates";
import Import from "./Import";
import Campaigns from "./Campaigns";
import Campaign from "./Campaign";
import Media from "./Media";

const { Content, Footer, Sider } = Layout;
const SubMenu = Menu.SubMenu;
const year = new Date().getUTCFullYear();

class Base extends React.Component {
  state = {
    basePath: "/" + window.location.pathname.split("/")[1],
    error: null,
    collapsed: false
  };

  onCollapse = collapsed => {
    this.setState({ collapsed });
  };

  render() {
    return (
      <Layout style={{ minHeight: "100vh" }}>
        <Sider
          collapsible
          collapsed={this.state.collapsed}
          onCollapse={this.onCollapse}
          theme="light"
        >
          <div className="logo">
            <Link to="/">
              <img src={logo} alt="listmonk logo" />
            </Link>
          </div>

          <Menu
            defaultSelectedKeys={["/"]}
            selectedKeys={[window.location.pathname]}
            defaultOpenKeys={[this.state.basePath]}
            mode="inline"
          >
            <Menu.Item key="/">
              <Link to="/">
                <Icon type="dashboard" />
                <span>Dashboard</span>
              </Link>
            </Menu.Item>
            <Menu.Item key="/lists">
              <Link to="/lists">
                <Icon type="bars" />
                <span>Lists</span>
              </Link>
            </Menu.Item>
            <SubMenu
              key="/subscribers"
              title={
                <span>
                  <Icon type="team" />
                  <span>Subscribers</span>
                </span>
              }
            >
              <Menu.Item key="/subscribers">
                <Link to="/subscribers">
                  <Icon type="team" /> All subscribers
                </Link>
              </Menu.Item>
              <Menu.Item key="/subscribers/import">
                <Link to="/subscribers/import">
                  <Icon type="upload" /> Import
                </Link>
              </Menu.Item>
            </SubMenu>

            <SubMenu
              key="/campaigns"
              title={
                <span>
                  <Icon type="rocket" />
                  <span>Campaigns</span>
                </span>
              }
            >
              <Menu.Item key="/campaigns">
                <Link to="/campaigns">
                  <Icon type="rocket" /> All campaigns
                </Link>
              </Menu.Item>
              <Menu.Item key="/campaigns/new">
                <Link to="/campaigns/new">
                  <Icon type="plus" /> Create new
                </Link>
              </Menu.Item>
              <Menu.Item key="/campaigns/media">
                <Link to="/campaigns/media">
                  <Icon type="picture" /> Media
                </Link>
              </Menu.Item>
              <Menu.Item key="/campaigns/templates">
                <Link to="/campaigns/templates">
                  <Icon type="code-o" /> Templates
                </Link>
              </Menu.Item>
            </SubMenu>
          </Menu>
        </Sider>

        <Layout>
          <Content style={{ margin: "0 16px" }}>
            <div className="content-body">
              <div id="alert-container" />
              <Switch>
                <Route
                  exact
                  key="/"
                  path="/"
                  render={props => (
                    <Dashboard {...{ ...this.props, route: props }} />
                  )}
                />
                <Route
                  exact
                  key="/lists"
                  path="/lists"
                  render={props => (
                    <Lists {...{ ...this.props, route: props }} />
                  )}
                />
                <Route
                  exact
                  key="/subscribers"
                  path="/subscribers"
                  render={props => (
                    <Subscribers {...{ ...this.props, route: props }} />
                  )}
                />
                <Route
                  exact
                  key="/subscribers/lists/:listID"
                  path="/subscribers/lists/:listID"
                  render={props => (
                    <Subscribers {...{ ...this.props, route: props }} />
                  )}
                />
                <Route
                  exact
                  key="/subscribers/import"
                  path="/subscribers/import"
                  render={props => (
                    <Import {...{ ...this.props, route: props }} />
                  )}
                />
                <Route
                  exact
                  key="/subscribers/:subID"
                  path="/subscribers/:subID"
                  render={props => (
                    <Subscriber {...{ ...this.props, route: props }} />
                  )}
                />
                <Route
                  exact
                  key="/campaigns"
                  path="/campaigns"
                  render={props => (
                    <Campaigns {...{ ...this.props, route: props }} />
                  )}
                />
                <Route
                  exact
                  key="/campaigns/new"
                  path="/campaigns/new"
                  render={props => (
                    <Campaign {...{ ...this.props, route: props }} />
                  )}
                />
                <Route
                  exact
                  key="/campaigns/media"
                  path="/campaigns/media"
                  render={props => (
                    <Media {...{ ...this.props, route: props }} />
                  )}
                />
                <Route
                  exact
                  key="/campaigns/templates"
                  path="/campaigns/templates"
                  render={props => (
                    <Templates {...{ ...this.props, route: props }} />
                  )}
                />
                <Route
                  exact
                  key="/campaigns/:campaignID"
                  path="/campaigns/:campaignID"
                  render={props => (
                    <Campaign {...{ ...this.props, route: props }} />
                  )}
                />
              </Switch>
            </div>
          </Content>
          <Footer>
            <span className="text-small">
              <a
                href="https://listmonk.app"
                rel="noreferrer noopener"
                target="_blank"
              >
                listmonk
              </a>{" "}
              &copy; 2019 {year !== 2019 ? " - " + year : ""} &mdash;{" "}
              <a
                href="https://listmonk.app/docs"
                target="_blank"
                rel="noopener noreferrer"
              >
                Docs
              </a>
            </span>
          </Footer>
        </Layout>
      </Layout>
    );
  }
}

export default Base;
