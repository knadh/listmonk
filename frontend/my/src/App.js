import React from 'react'
import Utils from './utils'
import { BrowserRouter } from 'react-router-dom'
import { notification } from "antd"
import axios from 'axios'
import qs from 'qs'

import Layout from './Layout'
import * as cs from './constants'

/*
  App acts as a an "automagic" wrapper for all sub components. It is also the central
  store for data required by various child components. In addition, all HTTP requests
  are fired through App.requests(), where successful responses are set in App's state
  for child components to access via this.props.data[type]. The structure is as follows:
    App.state.data = {
      "lists": [],
      "subscribers": []
      // etc.
    }

  A number of assumptions are made here for the "automagic" behaviour.
  1. All responses to resources return lists
  2. All PUT, POST, DELETE requests automatically append /:id to the API URIs.
*/

class App extends React.PureComponent {
    models = [cs.ModelUsers,
              cs.ModelSubscribers,
              cs.ModelLists,
              cs.ModelCampaigns,
              cs.ModelTemplates]

    state = {
        // Initialize empty states.
        reqStates: this.models.reduce((map, obj) => (map[obj] = cs.StatePending, map), {}),
        data: this.models.reduce((map, obj) => (map[obj] = [], map), {}),
        modStates: {}
    }

    componentDidMount = () => {
        axios.defaults.paramsSerializer = params => {
            return qs.stringify(params, {arrayFormat: "repeat"});
        }
    }

    // modelRequest is an opinionated wrapper for model specific HTTP requests,
    // including setting model states.
    modelRequest = async (model, route, method, params) => {
        let url = replaceParams(route, params)

        this.setState({ reqStates: { ...this.state.reqStates, [model]: cs.StatePending } })
        try {
            let req = {
                method: method,
                url: url,
            }

            if (method === cs.MethodGet || method === cs.MethodDelete) {
                req.params = params ? params : {}
            } else {
                req.data = params ? params : {}
            }

            
            let res = await axios(req)
            this.setState({ reqStates: { ...this.state.reqStates, [model]: cs.StateDone } })
            
            // If it's a GET call, set the response as the data state.
            if (method === cs.MethodGet) {
                this.setState({ data: { ...this.state.data, [model]: res.data.data } })
            }

            return res
        } catch (e) {
            // If it's a GET call, throw a global notification.
            if (method === cs.MethodGet) {
                notification["error"]({ message: "Error fetching data", description: Utils.HttpError(e).message, duration: 0 })
            }

            // Set states and show the error on the layout.
            this.setState({ reqStates: { ...this.state.reqStates, [model]: cs.StateDone } })
            throw Utils.HttpError(e)
        }
    }

    // request is a wrapper for generic HTTP requests.
    request = async (url, method, params, headers) => {
        url = replaceParams(url, params)

        this.setState({ reqStates: { ...this.state.reqStates, [url]: cs.StatePending } })
        try {
            let req = {
                method: method,
                url: url,
                headers: headers ? headers : {}
            }
            
            if(method === cs.MethodGet || method === cs.MethodDelete) {
                req.params =  params ? params : {}
            } else {
                req.data =  params ? params : {}
            }

            let res = await axios(req)

            this.setState({ reqStates: { ...this.state.reqStates, [url]: cs.StateDone } })
            return res
        } catch (e) {
            this.setState({ reqStates: { ...this.state.reqStates, [url]: cs.StateDone } })
            throw Utils.HttpError(e)
        }
    }


    pageTitle = (title) => {
        document.title = title
    }

    render() {
        return (
            <BrowserRouter>
                <Layout
                    modelRequest={ this.modelRequest }
                    request={ this.request }
                    reqStates={ this.state.reqStates }
                    pageTitle={ this.pageTitle }
                    config={ window.CONFIG }
                    data={ this.state.data } />
            </BrowserRouter>
        )
    }
}

function replaceParams (route, params) {
    // Replace :params in the URL with params in the array.
    let uriParams = route.match(/:([a-z0-9\-_]+)/ig)
    if(uriParams && uriParams.length > 0) {
        uriParams.forEach((p) => {
            let pName = p.slice(1) // Lose the ":" prefix
            if(params && params.hasOwnProperty(pName)) {
                route = route.replace(p, params[pName])
            }
        })
    }

    return route
}

export default App
