import Vue from 'vue'
import Router from 'vue-router'


import beginInterface from '@/components/beginInterface'

import bankInterface from '@/components/bank/bankInterface'
import checkAllBills from '@/components/bank/checkAllBills'
import issueBills from '@/components/bank/issueBills'
import dealWaitDiscountBills from '@/components/bank/dealWaitDiscountBills'
import queryHistory from '@/components/bank/queryHistory'


import companyInterface from '@/components/company/companyInterface'
import dealWaitPayBills from '@/components/company/dealWaitPayBills'
import checkAllWaitEndorseBills from '@/components/company/checkAllWaitEndorseBills'
import checkAllPayBills from '@/components/company/checkAllPayBills'
import checkAllAcceptBills from '@/components/company/checkAllAcceptBills'
import checkAllHoldBills from '@/components/company/checkAllHoldBills'


Vue.use(Router)

export default new Router({
  routes: [
    {
      path: '/queryHistory',
      name: 'queryHistory',
      component: queryHistory
    },

    {
      path: '/beginInterface',
      name: 'beginInterface',
      component: beginInterface
    },
    {
      path: '/bank/bankInterface',
      name: 'bankInterface',
      component: bankInterface
    },
    {
      path: '/bank/checkAllBills',
      name: 'checkAllBills',
      component: checkAllBills
    },
    {
      path: '/bank/issueBills',
      name: 'issueBills',
      component: issueBills
    },
    {
      path: '/bank/dealWaitDiscountBills',
      name: 'dealWaitDiscountBills',
      component: dealWaitDiscountBills
    },



    {
      path: '/company/companyInterface',
      name: 'companyInterface',
      component: companyInterface
    },
    {
      path: '/company/dealWaitPayBills',
      name: 'dealWaitPayBills',
      component: dealWaitPayBills
    },
    {
      path: '/company/checkAllWaitEndorseBills',
      name: 'checkAllWaitEndorseBills',
      component: checkAllWaitEndorseBills
    },
    {
      path: '/company/checkAllPayBills',
      name: 'checkAllPayBills',
      component: checkAllPayBills
    },
    {
      path: '/company/checkAllAcceptBills',
      name: 'checkAllAcceptBills',
      component: checkAllAcceptBills
    },
    {
      path: '/company/checkAllHoldBills',
      name: 'checkAllHoldBills',
      component: checkAllHoldBills
    },

  ]
})
