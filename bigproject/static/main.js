$(document).ready(function(){
    $('#filterinp').keyup(function(e){
        if(e.keyCode == 13 || $('#filterinp').val().length >= 3){
            updateTblData($('#filterinp').val());
        }
    });
    updateTblData("");
});
function updateTblData(filtertext){
    $('#tbldata').empty();
    $.get("http://localhost:12121/data?filter="+filtertext,function(result){
        var data = JSON.parse(result);
        if (data.length > 0){
            data.forEach(element => {
                $('#tbldata').append('<tr><td>'+element.ID+'</td>\
                                    <td>'+element.Name+'</td>\
                                    <td>'+element.MSISDN+'</td>\
                                    <td>'+element.Email+'</td>\
                                    <td>'+parseDate(element.BirthDate)+'</td>\
                                    <td>'+parseTime(element.Created)+'</td>\
                                    <td>'+parseTime(element.Update)+'</td>\
                                    <td>'+calcAge(element.BirthDate)+'</td></tr>');
            });
        }else{
            $('#tbldata').append('<tr><td colspan="8">No user found</td></tr>');
        }
    });
}
function parseDate(date){
    if (date != null){
        tmp = new Date(date);
        formatter = new Intl.NumberFormat('en-IN', { minimumIntegerDigits: 2, maximumFractionDigits:0, minimumFractionDigits:0 })
        return tmp.getFullYear()+'/'+addLeadingZeros(tmp.getMonth()+1,2)+'/'+addLeadingZeros(tmp.getDate()+1, 2);
    }
    return '-';
}
function parseTime(date){
    if (date != null){
        tmp = new Date(date);    
        return parseDate(date) + ' ' + addLeadingZeros(tmp.getHours(),2)+':'+addLeadingZeros(tmp.getMinutes(),2)+':'+addLeadingZeros(tmp.getSeconds(),2);
    }
    return '-';
}
function addLeadingZeros(number, digit){
    formatter = new Intl.NumberFormat('en-IN', { minimumIntegerDigits: digit, maximumFractionDigits:0, minimumFractionDigits:0 });
    return formatter.format(number);
}
function calcAge(birthdate){
    if (birthdate != null){
        var years = moment().diff(birthdate, 'years');
        return years;
    }
    return 0;
}