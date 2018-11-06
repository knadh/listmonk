import { Col, Row, notification, Card, Tooltip, Icon, Spin } from "antd"
import React from "react";
import { Chart, Axis, Geom, Tooltip as BizTooltip } from 'bizcharts';

import * as cs from "./constants"

class Dashboard extends React.PureComponent {
    state = {
        stats: null,
        loading: true
    }

    campaignTypes = ["running", "finished", "paused", "draft", "scheduled", "cancelled"]

    componentDidMount = () => {
        this.props.pageTitle("Dashboard")
        this.props.request(cs.Routes.GetDashboarcStats, cs.MethodGet).then((resp) => {
            this.setState({ stats: resp.data.data, loading: false })
        }).catch(e => {
            notification["error"]({ message: "Error", description: e.message })
            this.setState({ loading: false })
        })
    }

    orZero(v) {
        return v ? v : 0
    }
    
    render() {
        return (
            <section className = "dashboard">
                <h1>Welcome</h1>
                <hr />
                <Spin spinning={ this.state.loading }>
                { this.state.stats && 
                    <div className="stats">
                        <Row>
                            <Col span={ 16 }>
                                <Row gutter={ 24 }>
                                    <Col span={ 8 }>
                                        <Card title="Active subscribers" bordered={ false }>
                                            <h1 className="count">{ this.orZero(this.state.stats.subscribers.enabled) }</h1>
                                        </Card>
                                    </Col>
                                    <Col span={ 8 }>
                                        <Card title="Blacklisted subscribers" bordered={ false }>
                                            <h1 className="count">{ this.orZero(this.state.stats.subscribers.blacklisted) }</h1>
                                        </Card>
                                    </Col>
                                    <Col span={ 8 }>
                                        <Card title="Orphaned subscribers" bordered={ false }>
                                            <h1 className="count">{ this.orZero(this.state.stats.orphan_subscribers) }</h1>
                                        </Card>
                                    </Col>
                                </Row>
                            </Col>
                            <Col span={ 6 } offset={ 2 }>
                                <Row gutter={ 24 }>
                                    <Col span={ 12 }>
                                        <Card title="Public lists" bordered={ false }>
                                            <h1 className="count">{ this.orZero(this.state.stats.lists.public) }</h1>
                                        </Card>
                                    </Col>
                                    <Col span={ 12 }>
                                        <Card title="Private lists" bordered={ false }>
                                            <h1 className="count">{ this.orZero(this.state.stats.lists.private) }</h1>
                                        </Card>
                                    </Col>
                                </Row>
                            </Col>
                        </Row>
                        <hr />
                        <Row>
                            <Col span={ 16 }>
                                <Row gutter={ 24 }>
                                    <Col span={ 12 }>
                                        <Card title="Campaign views (last 3 months)" bordered={ false }>
                                            <h1 className="count">
                                                { this.state.stats.campaign_views.reduce((total, v) => total + v.count, 0) }
                                                {' '}
                                                views
                                            </h1>
                                            <Chart height={ 220 } padding={ [0, 0, 0, 0] } data={ this.state.stats.campaign_views } forceFit>
                                                <BizTooltip crosshairs={{ type : "y" }} />
                                                <Geom type="area" position="date*count" size={ 0 } color="#7f2aff" />
                                                <Geom type='point' position="date*count" size={ 0 } />
                                            </Chart>
                                        </Card>
                                    </Col>
                                    <Col span={ 12 }>
                                        <Card title="Link clicks (last 3 months)" bordered={ false }>
                                            <h1 className="count">
                                                { this.state.stats.link_clicks.reduce((total, v) => total + v.count, 0) }
                                                {' '}
                                                clicks
                                            </h1>
                                            <Chart height={ 220 } padding={ [0, 0, 0, 0] } data={ this.state.stats.link_clicks } forceFit>
                                                <BizTooltip crosshairs={{ type : "y" }} />
                                                <Geom type="area" position="date*count" size={ 0 } color="#7f2aff" />
                                                <Geom type='point' position="date*count" size={ 0 } />
                                            </Chart>
                                        </Card>
                                    </Col>
                                </Row>
                            </Col>

                            <Col span={ 6 } offset={ 2 }>
                                <Card title="Campaigns" bordered={ false } className="campaign-counts">
                                    { this.campaignTypes.map((key) =>
                                        <Row key={ `stats-campaigns-${ key }` }>
                                            <Col span={ 18 }><h1 className="name">{ key }</h1></Col>
                                            <Col span={ 6 }>
                                                <h1 className="count">
                                                    { this.state.stats.campaigns.hasOwnProperty(key) ?
                                                        this.state.stats.campaigns[key] : 0 }
                                                </h1>
                                            </Col>
                                        </Row>
                                    )}
                                </Card>
                            </Col>
                        </Row>
                    </div>
                }
                </Spin>
            </section>
        );
    }
}

export default Dashboard;
