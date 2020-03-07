import React from "react"
import {
  Row,
  Col,
  Checkbox,
} from "antd"

import * as cs from "./constants"

class Forms extends React.PureComponent {
  state = {
    lists: [],
    selected: []
  }

  componentDidMount() {
    this.props.pageTitle("Forms")
    this.props
      .modelRequest(cs.ModelLists, cs.Routes.GetLists, cs.MethodGet, {
        per_page: "all"
      })
      .then(() => {
        this.setState({ lists: this.props.data[cs.ModelLists].results })
      })
  }

  handleSelection(sel) {
    let out = []
    sel.forEach(s => {
      const item = this.state.lists.find(l => {
        return l.uuid === s
      })
      if (item) {
        out.push(item)
      }
    })

    console.log(out)
    this.setState({ selected: out })
  }

  render() {
    return (
      <section className="content list-form">
        <h1>Forms</h1>
        <Row>
          <Col span={8}>
            <Checkbox.Group
              className="lists"
              options={this.state.lists.map(l => {
                return { label: l.name, value: l.uuid }
              })}
              defaultValue={[]}
              onChange={(sel) => this.handleSelection(sel)}
            />
          </Col>
          <Col span={16}>
              <h1>Form HTML</h1>
              <p>Use the following HTML to show a subscription form on an external webpage.</p>
              <p>
                The form should have the <code><strong>email</strong></code> field and one or more{" "}
                <code><strong>l</strong></code> (list UUID) fields. The <code><strong>name</strong></code> field is optional.
              </p>
            <pre className="html">

{`<form method="post" action="${window.CONFIG.rootURL}/subscription/form" class="listmonk-subscription">
    <div>
        <h3>Subscribe</h3>
        <p><input type="text" name="email" value="" placeholder="E-mail" /></p>
        <p><input type="text" name="name" value="" placeholder="Name (optional)" /></p>`}
{(() => {
    let out = [];
    this.state.selected.forEach(l => {
        out.push(`
        <p>
            <input type="checkbox" name="l" value="${l.uuid}" id="${l.uuid.substr(0,5)}" />
            <label for="${l.uuid.substr(0,5)}">${l.name}</label>
        </p>`);
    });
    return out;
})()}
{`
        <p><input type="submit" value="Subscribe" /></p>
    </div>
</form>
`}

            </pre>
          </Col>
        </Row>
      </section>
    )
  }
}

export default Forms
