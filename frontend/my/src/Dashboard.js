import { Button, Col, Form, Icon, Input, Modal, notification, Popconfirm, Row, Select, Spin, Table, Tag, Tooltip } from "antd"
import React from "react";

import * as cs from "./constants"

class Dashboard extends React.PureComponent {
    state = {
        stats: null
    }

    componentDidMount = () => {
        this.props.pageTitle("Dashboard")

        this.props.request(cs.Routes.GetDashboarcStats, cs.MethodGet).then((resp) => {
            this.setState({ stats: resp.data.data })
        }).catch(e => {
            notification["error"]({ message: "Error", description: e.message })
        })
    }
    
    render() {
        return (
            <section className = "dashboard">
                <h1>Welcome</h1>

                { this.state.stats && 
                    <div className="stats">
                        <Row>
                            <Col span={ 12 }>
                                <h1></h1>
                            </Col>
                        </Row>
                    </div>
                }
            </section>
        );
    }
}

export default Dashboard;
