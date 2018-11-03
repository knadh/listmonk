import React from 'react';

class Dashboard extends React.PureComponent {
  componentDidMount = () => {
    this.props.pageTitle("Dashboard")
  }
  render() {
    return (
        <h1>Welcome</h1>
    );
  }
}

export default Dashboard;
